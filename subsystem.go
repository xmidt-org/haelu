// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"encoding/json"
	"iter"
	"time"
)

// Updater is a interface that can be used to update a subsystem's
// health Status as well as pause and resume monitoring.
type Updater interface {
	// Update supplies a possibly new status and an optional error that
	// occurred while checking the status or otherwise using the subsystem.
	Update(Status, error)
}

// Name is the human-readable identifier for a subsystem.  Names must be
// unique within a Monitor.
type Name string

// Definition holds the information necessary to create a logical subsystem
// within a Monitor.
type Definition struct {
	// Name is the unique identifier for this subsystem within the Monitor.
	Name Name

	// Status is the initial status for this subsystem. By default, a subsystem's
	// initial state is StatusGood. Use this field to set a different Status that
	// will have effect until the first time an update occurs.
	Status Status

	// NonCritical indicates how this subsystem affects the overall Monitor status. By
	// default, this field is false, which means that a subsystem is critical.
	//
	// A critical subsystem directly affects a Monitor's overall status. If any critical
	// subsystems are StatusWarn, the overall status will be StatusWarn. If any critical
	// subsystems are StatusBad, the overall status will be StatusBad.
	//
	// A noncritcal subsystem will never cause a Monitor to be StatusBad. If any noncritical
	// subsystem is NOT StatusGood, the overall status will be StatusWarn.
	NonCritical bool

	// Probe is an optional closure that interrogates this subsystem's state. A probe will be
	// called only if both (1) the Monitor has been started, and (2) the subsystem is not paused.
	//
	// If no Probe is specified, the only way to update a subsystem's state is via its Updater.
	Probe Probe

	// ProbeInterval is the time interval on which any Probe is invoked. If no Probe is set,
	// this field is ignored.
	ProbeInterval time.Duration

	// Metadata are optional name/value pairs to associate with this subsystem. A caller may
	// specify any values in this map to act as metadata for the subsystem.
	Metadata Metadata
}

// Subsystem is a snapshot of the current state of a logical subsystem within a monitor.
type Subsystem struct {
	// Name is the unique identifier for this subsystem.
	Name Name `json:"name" yaml:"name"`

	// Status is the current status of this subsystem. When creating a
	// Monitor, this is the initial status.
	Status Status `json:"status" yaml:"status"`

	// LastUpdate is the UTC timestamp of the last status update to this subsystem.
	// This field is set to the current time upon creation. Only status updates
	// affect this timestamp. Pausing or disabling a subsystem does not update
	// this field.
	LastUpdate time.Time `json:"lastUpdate,omitempty" yaml:"lastUpdate"`

	// LastError is the error that occurred with the most recent status update, if any.
	// This field is ignored when creating a Monitor.
	LastError error `json:"lastError,omitempty" yaml:"lastError"`

	// NonCritical indicates whether this subsystem is noncritical, i.e. how it affects
	// the overall Monitor status.
	NonCritical bool `json:"nonCritical" yaml:"nonCritical"`

	// Metadata is the optional set of name/value pairs that were supplied when the
	// subsystem was defined.
	Metadata Metadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Subsystems is an immutable, iterable sequence of Subsystem snapshots.
type Subsystems struct {
	ss []Subsystem
}

// AsSubsystems creates an immutable Subsystems sequence from a slice
// of individual Subsystem instances. The returned Subsystems will be
// a shallow copy of the given slice.
//
// If the subs slice is empty, the returned Subsystems will be an
// immutable, empty sequence.
func AsSubsystems(subs ...Subsystem) (s Subsystems) {
	// NOTE: wrap a nil slice in the returned Subsystems if
	// the original slice is empty.
	if len(subs) > 0 {
		s.ss = make([]Subsystem, len(subs))
		copy(s.ss, subs)
	}

	return
}

// Len returns the count of Subsystem snapshots in this sequence.
func (s Subsystems) Len() int {
	return len(s.ss)
}

// Get returns the Subsystem at the given 0-based index. If i is
// negative or not less than Len(), this function panics.
func (s Subsystems) Get(i int) Subsystem {
	return s.ss[i]
}

// All provides an iterator over this immutable sequence.
func (s Subsystems) All() iter.Seq[Subsystem] {
	return func(f func(Subsystem) bool) {
		for _, s := range s.ss {
			if !f(s) {
				return
			}
		}
	}
}

// MarshalJSON marshals this sequence as a slice of Subsystems.
func (s Subsystems) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ss)
}
