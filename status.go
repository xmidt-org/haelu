// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

//go:generate stringer -type=Status -linecomment

// Status indicates the health status of a single subsystem or the overall application.
type Status uint8

const (
	// StatusGood indicates a healthy application or subsystem.
	StatusGood Status = iota // good

	// StatusWarn indicates an application or subsystem that is usable, but is having problems.
	StatusWarn // warn

	// StatusBad indicates an application or subystem that is completely unusable.
	StatusBad // bad
)

// MarshalText produces the string value of this Status.
func (s Status) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}
