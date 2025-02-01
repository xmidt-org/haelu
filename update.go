// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

// Updater is a interface that can be used to update a subsystem's
// health Status as well as pause and resume monitoring.
type Updater interface {
	// Update supplies a possibly new status and an optional error that
	// occurred while checking the status or otherwise using the subsystem.
	Update(Status, error)
}
