// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"iter"
	"time"
)

// MonitorEvent indicates a change in the state of a Monitor. A MonitorEvent is dispatched
// when Start is called, to indicate the initial state of the Monitor and its subsystems.
//
// A MonitorEvent is also dispatched anytime a subsystem is updated, even if the Status
// didn't change. This is because an Update can contain other information that may have
// changed, such as the Attributes. The StatusChanged field can be used to determine if
// the overall status has actually changed from the last event.
type MonitorEvent struct {
	// Status is the new overall status of the Monitor.
	Status Status

	// LastUpdate is the timestamp of the monitor's last update to any subsystem.
	// This will include updates that may not have changed the status.
	//
	// This timestamp will always be in UTC.
	LastUpdate time.Time

	// SubsystemCount is the count of subsystems that will be returned by
	// the Subsystems iterator. This is a useful hint for callers that need
	// to make a copy of the sequence.
	SubsystemCount int

	// Subsystems is a snapshot of the state of each subsystem within
	// the Monitor that fired this event. Even if the Monitor has no
	// subsystems defined, this field is not nil, but will instead be
	// a sequence of zero (0) elements.
	//
	// The set of subsystems is immutable.  Callers can use this sequence to
	// iterate over them in an atomic fashion. This iterator is only valid
	// until the listener returns from its event method. If a caller needs to retain
	// the set of subsystems, it must make a copy.
	Subsystems iter.Seq[Subsystem]
}

// GetSubsystems returns a distinct copy of the Subsystems iterator. This handles the
// most common use case that callers need.
func (me MonitorEvent) GetSubsystems() (ss []Subsystem) {
	ss = make([]Subsystem, 0, me.SubsystemCount)
	for s := range me.Subsystems {
		ss = append(ss, s)
	}

	return
}

// MonitorListener is a sink for MonitorEvents.
type MonitorListener interface {
	// OnMonitorEvent receives a MonitorEvent. This method
	// must not panic or block. Additionally, this method
	// must not invoke any Monitor methods, as event dispatch
	// is executed under the Monitor's internal lock so that
	// listeners get an atomically update Status.
	OnMonitorEvent(MonitorEvent)
}

// MonitorListeners is an aggregate MonitorListener.
type MonitorListeners []MonitorListener

// OnMonitorEvent dispatches the given event to each listener
// in this aggregate.
func (mls MonitorListeners) OnMonitorEvent(e MonitorEvent) {
	for _, l := range mls {
		l.OnMonitorEvent(e)
	}
}
