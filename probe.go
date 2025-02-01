// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"context"
	"reflect"
	"time"
)

const (
	// DefaultProbeInterval is the interval a Monitor will invoke a Probe on
	// when no interval is set. A Monitor may have its own default, which can
	// be set via WithDefaultProbeInterval.
	DefaultProbeInterval time.Duration = 2 * time.Minute
)

// Probe is a callback type to interrogate a subsystem for its health status.
// A Probe may consult information out-of-process, so it's passed a context.Context
// that gets canceled when a Monitor is shutdown.
type Probe func(context.Context) (Status, error)

// ProbeFunc describes the various closure types that are convertible to Probes.
// Calling code can convert any closure that satisfies this type via AsProbe.
type ProbeFunc interface {
	~func() bool |
		~func(context.Context) bool |
		~func() error |
		~func(context.Context) error |
		~func() Status |
		~func(context.Context) Status |
		~func() (Status, error) |
		~func(context.Context) (Status, error)
}

var (
	probeReturnBool        = reflect.TypeOf((func() bool)(nil))
	probeContextReturnBool = reflect.TypeOf((func(context.Context) bool)(nil))

	probeReturnError        = reflect.TypeOf((func() error)(nil))
	probeContextReturnError = reflect.TypeOf((func(context.Context) error)(nil))

	probeReturnStatus        = reflect.TypeOf((func() Status)(nil))
	probeContextReturnStatus = reflect.TypeOf((func(context.Context) Status)(nil))

	probeReturnStatusError        = reflect.TypeOf((func() (Status, error))(nil))
	probeContextReturnStatusError = reflect.TypeOf((func(context.Context) (Status, error))(nil))
)

// AsProbe converts a closure into a Probe. This allows client code to use
// simpler closures that have no dependency on this package.
//
// For closures that return a simple error, ErrorStatus is used to determine
// the health Status of the probe.
func AsProbe[F ProbeFunc](f F) Probe {
	fv := reflect.ValueOf(f)
	switch {
	case fv.CanConvert(probeReturnBool):
		pf := fv.Convert(probeReturnBool).Interface().(func() bool)
		return func(_ context.Context) (Status, error) {
			if pf() {
				return StatusGood, nil
			} else {
				return StatusBad, nil
			}
		}

	case fv.CanConvert(probeContextReturnBool):
		pf := fv.Convert(probeContextReturnBool).Interface().(func(context.Context) bool)
		return func(ctx context.Context) (Status, error) {
			if pf(ctx) {
				return StatusGood, nil
			} else {
				return StatusBad, nil
			}
		}

	case fv.CanConvert(probeReturnError):
		pf := fv.Convert(probeReturnError).Interface().(func() error)
		return func(_ context.Context) (Status, error) {
			err := pf()
			return ErrorStatus(err), err
		}

	case fv.CanConvert(probeContextReturnError):
		pf := fv.Convert(probeContextReturnError).Interface().(func(context.Context) error)
		return func(ctx context.Context) (Status, error) {
			err := pf(ctx)
			return ErrorStatus(err), err
		}

	case fv.CanConvert(probeReturnStatus):
		pf := fv.Convert(probeReturnStatus).Interface().(func() Status)
		return func(_ context.Context) (Status, error) {
			return pf(), nil
		}

	case fv.CanConvert(probeContextReturnStatus):
		pf := fv.Convert(probeContextReturnError).Interface().(func(context.Context) Status)
		return func(ctx context.Context) (Status, error) {
			return pf(ctx), nil
		}

	case fv.CanConvert(probeReturnStatusError):
		pf := fv.Convert(probeReturnStatusError).Interface().(func() (Status, error))
		return func(_ context.Context) (Status, error) {
			return pf()
		}

	default: // this is the exact signature of the Probe type
		return fv.Convert(probeContextReturnStatusError).Interface().(func(context.Context) (Status, error))
	}
}
