// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import "time"

// now is a closure used to produce the current time.
// By default, time.Now is used.
type now func() time.Time

// newTimer is a factory closure for a timer channel and the associated Stop function.
type newTimer func(time.Duration) (<-chan time.Time, func() bool)

// defaultNewTimer is the default enewTimer closure used to produce
// a timer channel and stop function.
func defaultNewTimer(d time.Duration) (<-chan time.Time, func() bool) {
	t := time.NewTimer(d)
	return t.C, t.Stop
}
