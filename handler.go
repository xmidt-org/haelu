// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	// HandlerInitialResponseBody is the plaintext message that a Handler writes when there
	// has been no update yet.
	HandlerInitialResponseBody = "no health status update received yet"
)

// HealthResponseCoder is a strategy for turning a health Status into an HTTP response code.
type HealthResponseCoder func(Status) int

// DefaultHealthResponseCoder is the default HealthResponseCoder used when no
// strategy is supplied.
//
// This function returns a 200 for StatusGood, 429 for StatusWarn (consul's convention),
// and a 500 for any other status.
func DefaultHealthResponseCoder(s Status) int {
	switch s {
	case StatusGood:
		return http.StatusOK

	case StatusWarn:
		return http.StatusTooManyRequests

	default:
		return http.StatusInternalServerError
	}
}

// Errorer is a callback that receives errors the Handler encounters while
// trying to write requests. By default, such errors are dropped.
type Errorer func(error)

// HandlerOption is a configurable option for customizing a health Handler.
type HandlerOption interface {
	apply(*Handler) error
}

type handlerOptionFunc func(*Handler) error

func (f handlerOptionFunc) apply(h *Handler) error { return f(h) }

// WithHealthResponseCoder sets a custom strategy for determining the HTTP response code
// for a given health Status.
//
// If this option isn't used or is set to nil, DefaultHealthResponseCoder is used.
func WithHealthResponseCoder(f HealthResponseCoder) HandlerOption {
	return handlerOptionFunc(func(h *Handler) error {
		h.coder = f
		return nil
	})
}

// WithErrorer configures an error callback for the Handler. There is
// no default for this option. If unspecified, errors are dropped.
func WithErrorer(errorer Errorer) HandlerOption {
	return handlerOptionFunc(func(h *Handler) error {
		h.errorer = errorer
		return nil
	})
}

// content holds the marshaled content held by a handler. This represents
// prerendered content that can be replayed for each request.
type content struct {
	// responseCode is the HTTP response code to return. This will be
	// determined by the HealthResponseCoder.
	responseCode int

	// contentType is the HTTP media type of the body.
	contentType string

	// contentLength is the string value of the body's length. This is
	// cached here to avoid doing excessive int-to-string conversions.
	contentLength string

	// lastModified is the Last-Modified header value, which is the
	// lastUpdate field formatted using http.TimeFormat.
	lastModified string

	// lastUpdate is the UTC time that this cached content was updated.
	lastUpdate time.Time

	// body is the HTTP body for this content type.
	body []byte
}

// writeTo writes this prerendered content to the given response.
func (c content) writeTo(response http.ResponseWriter) (err error) {
	rh := response.Header()
	rh.Set("Content-Type", c.contentType)
	rh.Set("Content-Length", c.contentLength)
	rh.Set("Last-Modified", c.lastModified)

	response.WriteHeader(c.responseCode)
	_, err = response.Write(c.body)
	return
}

// Handler is an HTTP handler that exposes health status. A Handler is
// a MonitorListener and receives the status it reports through events.
type Handler struct {
	coder   HealthResponseCoder
	errorer Errorer
	content atomic.Value
}

// NewHandler constructs a new health Handler using the supplied set of options.
// After construction, a Handler must be registered as a listener for a Monitor.
//
// Before any health updates are received, the returned handle will return
// http.StatusServiceUnavailable.
func NewHandler(opts ...HandlerOption) (*Handler, error) {
	h := new(Handler)
	for _, o := range opts {
		if err := o.apply(h); err != nil {
			return nil, err
		}
	}

	if h.coder == nil {
		h.coder = DefaultHealthResponseCoder
	}

	// initialize the content
	h.updateContent(
		time.Now().UTC(),
		http.StatusServiceUnavailable,
		"text/plain; charset=utf-8",
		[]byte(HandlerInitialResponseBody),
	)

	return h, nil
}

// ServeHTTP returns an HTTP response that represents the most recent health update.
func (h *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// force clients to always revalidate and fetch the current value
	response.Header().Set("Cache-Control", "no-cache")

	err := h.content.Load().(content).writeTo(response)
	if err != nil && h.errorer != nil {
		h.errorer(err)
	}
}

// marshalJSON marshals the event as a JSON message.
func (h *Handler) marshalJSON(e MonitorEvent) ([]byte, error) {
	return json.Marshal(
		struct {
			Status     Status      `json:"status"`
			LastUpdate time.Time   `json:"lastUpdate"`
			Subsystems []Subsystem `json:"subsystem"`
		}{
			Status:     e.Status,
			LastUpdate: e.LastUpdate,
			Subsystems: e.GetSubsystems(),
		},
	)
}

// updateContent updates this Handler's cached content for a given type. Currently, only
// JSON marshaling is supported, with textual content for errors.
func (h *Handler) updateContent(lastUpdate time.Time, responseCode int, contentType string, body []byte) {
	h.content.Store(
		content{
			responseCode:  responseCode,
			contentType:   contentType,
			contentLength: strconv.Itoa(len(body)),
			lastModified:  lastUpdate.Format(http.TimeFormat),
			lastUpdate:    lastUpdate,
			body:          body,
		},
	)
}

// updateError updates the cached content for an error, usually a
// marshaling error.
func (h *Handler) updateError(lastUpdate time.Time, err error) {
	h.updateContent(
		lastUpdate,
		http.StatusInternalServerError,
		"text/plain; charset=utf-8",
		[]byte(err.Error()),
	)
}

// OnMonitorEvent updates this handler's internal state.
func (h *Handler) OnMonitorEvent(e MonitorEvent) {
	var (
		lastUpdate   = e.LastUpdate
		responseCode = h.coder(e.Status)
	)

	if data, err := h.marshalJSON(e); err == nil {
		h.updateContent(
			lastUpdate,
			responseCode,
			"application/json",
			data,
		)
	} else {
		h.updateError(lastUpdate, err)
	}
}
