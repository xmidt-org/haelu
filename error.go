// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import "errors"

// SelfStatuser is an optional interface that an error can implement
// to indicate its health status.
type SelfStatuser interface {
	Status() Status
}

// SelfBooler is an alternative to SelfStatuser that lets an error simply
// indicate good or bad health.
type SelfBooler interface {
	Status() bool
}

type statusError struct {
	err    error
	status Status
}

func (se *statusError) Error() string {
	return se.err.Error()
}

func (se *statusError) Unwrap() error {
	return se.err
}

// AddStatus associates a health Status with the given error. The
// returned error will wrap err and implement SelfStatuser.
//
// If err already has a status associated with it, it will be
// replaced with the given status.
func AddStatus(err error, status Status) error {
	return &statusError{
		err:    err,
		status: status,
	}
}

// ErrorStatus examines an error to determine what health Status to
// associated with it.
//
// If err is nil, this function returns StatusGood.
//
// If err implements SelfStatuser, then the result of SelfStatuser.Status() is returned.
//
// If err implements SelfBooler, then StatusGood or StatusBad is returned
// based on the return value of SelfBooler.Status().
//
// For a non-nil error that does not implement one of the optional
// interfaces in this package, this function returns StatusBad.
func ErrorStatus(err error) Status {
	var (
		s SelfStatuser
		b SelfBooler
	)

	switch {
	case err == nil:
		return StatusGood

	case errors.As(err, &s):
		return s.Status()

	case errors.As(err, &b):
		if b.Status() {
			return StatusGood
		} else {
			return StatusBad
		}

	default:
		return StatusBad
	}
}
