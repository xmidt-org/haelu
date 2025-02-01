// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import "time"

// Attributes are a set of name/value pairs associated with a Subsystem.
// Attributes are not used by a health Monitor and can provide extra
// information about the subsystem for reporting.
type Attributes map[string]any

// Clone creates a shallow copy of this Attributes. Individual values
// are transferred as is to the clone. If this Attributes is empty,
// a nil Attributes is returned by this method.
//
// In general, callers should not retain Attributes in a definition or
// from an event. If Attributes need to be retained, use this method to
// make a copy. Note that any mutable values would still need to be copied.
func (a Attributes) Clone() Attributes {
	if len(a) == 0 {
		return nil
	}

	clone := make(Attributes, len(a))
	for k, v := range a {
		clone[k] = v
	}

	return clone
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

	// Attributes are optional name/value pairs to associate with this subsystem. A caller may
	// specify any values in this map. The Monitor does not modify this field, but does make
	// a shallow copy of these Attributes for its internal storage.
	Attributes Attributes
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

	// Attributes is the optional set of name/value pairs that were supplied when the
	// subsystem was defined.
	Attributes Attributes `json:"attributes,omitempty" yaml:"attributes,omitempty"`
}
