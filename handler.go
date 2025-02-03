// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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

// WithMonitor sets the health Monitor the Handler uses when
// presenting status.
func WithMonitor(m *Monitor) HandlerOption {
	return handlerOptionFunc(func(h *Handler) error {
		h.monitor = m
		return nil
	})
}

// Handler is an HTTP handler that exposes health status. A Handler uses
// a Monitor's State to render HTTP responses.
type Handler struct {
	coder   HealthResponseCoder
	monitor *Monitor
}

// NewHandler constructs a new health Handler using the supplied set of options.
func NewHandler(opts ...HandlerOption) (*Handler, error) {
	h := new(Handler)
	for _, o := range opts {
		if err := o.apply(h); err != nil {
			return nil, err
		}
	}

	if h.monitor == nil {
		return nil, errors.New("no monitor configured")
	}

	if h.coder == nil {
		h.coder = DefaultHealthResponseCoder
	}

	return h, nil
}

// ServeHTTP returns an HTTP response that represents the most recent health update.
func (h *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// force clients to always revalidate and fetch the current value
	response.Header().Set("Cache-Control", "no-cache")
	state := h.monitor.State()
	data, err := json.Marshal(state)

	if err == nil {
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("Content-Length", strconv.Itoa(len(data)))
		response.Header().Set("Last-Modified", state.LastUpdate.Format(http.TimeFormat))
		response.WriteHeader(h.coder(state.Status))
		_, err = response.Write(data)
	}

	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}
