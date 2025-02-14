// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"time"

	"github.com/xmidt-org/chronon"
)

// fakeTimer creates a fake, controllable newTimer closure
// from the given FakeClock.
func fakeTimer(fc *chronon.FakeClock) newTimer {
	return func(d time.Duration) (<-chan time.Time, func() bool) {
		ft := fc.NewTimer(d)
		return ft.C(), ft.Stop
	}
}
