// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrMonitorStarted is returned by Monitor.Start to indicate that the Monitor
	// has already been started.
	ErrMonitorStarted = errors.New("the monitor has been started")

	// ErrMonitorShutdown is returned by Monitor.Shutdown to indicate that the
	// Monitor has not yet been started or has already been Shutdown.
	ErrMonitorShutdown = errors.New("the monitor has been shutdown")
)

// MonitorState holds a snapshot of the state of a Monitor.
type MonitorState struct {
	// Status is the new overall status of the Monitor.
	Status Status `json:"status" yaml:"status"`

	// LastUpdate is the timestamp of the monitor's last update to any subsystem.
	// This will include updates that may not have changed the status.
	//
	// This timestamp will always be in UTC.
	LastUpdate time.Time `json:"lastUpdate" yaml:"lastUpdate"`

	// Subsystems is a snapshot of the state of each subsystem within
	// the Monitor.
	Subsystems Subsystems `json:"subsystems" yaml:"subsystems"`
}

// subsystemTracker holds all the information for tracking the state of
// a single subsystem.
type subsystemTracker struct {
	// lock is "inherited" from the containing monitor
	lock sync.Locker

	// now is the current time strategy "inherited" from the containing monitor
	now now

	// newTimer is the timer strategy "inherited" from the containing monitor
	newTimer newTimer

	// unsafeUpdateState is the "inherited" non-atomic closure that updates monitor
	// state.
	unsafeUpdateState func(time.Time)

	// definition is the configuration used to create this subsystem
	definition Definition

	// current represents the current state of this monitor.  This is a pointer
	// into an element of the Monitor's subsystems.
	current *Subsystem
}

// initialize sets up this tracker's initial state, using both its definition
// and information from the containing Monitor.
func (sst *subsystemTracker) initialize(m *Monitor, initialLastUpdate time.Time, current *Subsystem) {
	sst.now = m.now
	sst.newTimer = m.newTimer

	// take the initial state from the definition
	sst.current = current
	sst.current.Name = sst.definition.Name
	sst.current.Status = sst.definition.Status
	sst.current.NonCritical = sst.definition.NonCritical
	sst.current.Metadata = sst.definition.Metadata
	sst.current.LastUpdate = initialLastUpdate

	// normalize the probe interval
	if sst.definition.Probe == nil {
		sst.definition.ProbeInterval = 0
	} else if sst.definition.ProbeInterval <= 0 {
		sst.definition.ProbeInterval = m.defaultProbeInterval
	}
}

// startProbeTask ensures that a background goroutine is running
// to monitor the results from a Probe. If this subsystem has no Probe,
// this method does nothing.
//
// If this method starts a goroutine, it will stop with the supplied
// context is canceled.
func (sst *subsystemTracker) startProbeTask(ctx context.Context) {
	if sst.definition.Probe == nil {
		return
	}

	go func() {
		for {
			timeCh, stop := sst.newTimer(sst.definition.ProbeInterval)
			select {
			case <-ctx.Done():
				stop()
				return

			case <-timeCh:
				s, err := sst.definition.Probe(ctx)
				sst.Update(s, err)
			}
		}
	}()
}

// Update implements the Updater interface. This method updates this
// tracker's state under the monitor's lock. It then invokes the
// unsafeUpdateState closure to allow the monitor to update its
// overall status.
func (sst *subsystemTracker) Update(s Status, err error) {
	defer sst.lock.Unlock()
	sst.lock.Lock()

	sst.current.Status = s
	sst.current.LastError = err
	sst.current.LastUpdate = sst.now().UTC()

	sst.unsafeUpdateState(sst.current.LastUpdate)
}

// Monitor is a health status monitor for application subsystems.
// All methods on a Monitor are atomic.
//
// Each subsystem in a Monitor can be updated in one of two ways:
//
// (1) After construction, Get can be used to obtain an Updater for
// a subsystem. This Updater can be used at any time to update the status
// of a subsystem, which will cause the overall status of the Monitor
// to be recomputed.
//
// (2) A subsystem can be defined with a Probe. This Probe is a callback
// that will be invoked on the configured interval. Each time a Probe returns
// a result, that Probe's subsystem is update and the overall status of
// the Monitor is recomputed.
type Monitor struct {
	defaultProbeInterval time.Duration

	// now is the strategy used to get the current time.
	// by default, time.Now is used.
	now now

	// newTimer is a factory for creating the timer channel and stop function.
	// if unset, defaultNewTimer is used.
	//
	// Tests can replace this function to control probe monitoring.
	newTimer newTimer

	byName     map[Name]*subsystemTracker
	trackers   []*subsystemTracker
	subsystems []Subsystem

	// lock is primarily used to guard subsystem updates
	lock sync.Mutex

	// state is the overall state of this Monitor
	state atomic.Value

	// cancel is the cancellation function used to control any probe tasks
	cancel context.CancelFunc
}

// unsafeUpdateState performs the following:
//
// (1) computes the (possibly) new overall status based on the current subystem states
// (2) updates the atomic state for this Monitor
//
// The timestamp of the update is supplied so that it's consistent with the timestamp
// of any individual subsystem updates.
//
// This method must be executed under the monitor lock or in a situation where no
// concurrent invocation is possible.
func (m *Monitor) unsafeUpdateState(timestamp time.Time) {
	var (
		overall           Status
		criticalStatus    Status
		nonCriticalStatus Status
	)

	for _, st := range m.trackers {
		switch {
		case st.definition.NonCritical && st.current.Status > nonCriticalStatus:
			nonCriticalStatus = st.current.Status

		case !st.definition.NonCritical && st.current.Status > criticalStatus:
			criticalStatus = st.current.Status
		}
	}

	switch {
	case criticalStatus != StatusGood:
		overall = criticalStatus

	case nonCriticalStatus != StatusGood:
		overall = StatusWarn

	default:
		overall = StatusGood
	}

	m.state.Store(MonitorState{
		Status:     overall,
		LastUpdate: timestamp,
		Subsystems: AsSubsystems(m.subsystems...),
	})
}

// Len returns the count of subsystems that are defined for this Monitor.
func (m *Monitor) Len() int {
	return len(m.trackers)
}

// Get returns the Updater for a Subsystem. If no such Subsystem exists,
// this method returns (nil, false).
//
// This method always returns the same Updater instance for a given subsystem.
// The returned Updater may be used at any time, including when the Monitor
// has not been started or has been shutdown.
func (m *Monitor) Get(n Name) (Updater, error) {
	// no locking necessary, as the set of subsystems is immutable
	updater := m.byName[n]
	if updater == nil {
		return nil, fmt.Errorf("No subsystem with the name [%s] is registered", n)
	}

	return updater, nil
}

// State returns the last computed state for this Monitor.
func (m *Monitor) State() MonitorState {
	return m.state.Load().(MonitorState)
}

// Start computes the initial, overall state based on the status of the subystems
// and then starts any background tasks to monitor subsystem Probes. A Monitor may
// receive updates from subsystems at any time, even before Start is called.
//
// This method is idempotent. If this Monitor has already been started, this method
// does nothing and returns ErrMonitorStarted.
//
// Start will update the overall timestamp for the State, but will not modify any
// LastUpdate fields for subsystems.
func (m *Monitor) Start() error {
	defer m.lock.Unlock()
	m.lock.Lock()

	if m.cancel != nil {
		return ErrMonitorStarted
	}

	m.unsafeUpdateState(m.now().UTC())
	var rootCtx context.Context
	rootCtx, m.cancel = context.WithCancel(context.Background())
	for _, st := range m.trackers {
		st.startProbeTask(rootCtx)
	}

	return nil
}

// Shutdown stops any running tasks. The status of subsystems are preserved.
// After this method has been called, Probes are no longer run but any
// Updaters may still be used to update subsystem states.
//
// This method is idempotent. If this Monitor is not running,
// this method does nothing and returns ErrMonitorShutdown.
func (m *Monitor) Shutdown() error {
	defer m.lock.Unlock()
	m.lock.Lock()

	if m.cancel == nil {
		return ErrMonitorShutdown
	}

	m.cancel()
	m.cancel = nil
	return nil
}

// MonitorOption is a configurable option for tailoring a Monitor.
type MonitorOption interface {
	apply(*Monitor) error
}

type monitorOptionFunc func(*Monitor) error

func (f monitorOptionFunc) apply(m *Monitor) error { return f(m) }

// WithDefaultProbeInterval sets the default interval for invoking any
// registered probes for this Monitor. If unset or nonpositive,
// the Monitor will use DefaultProbeInterval.
func WithDefaultProbeInterval(i time.Duration) MonitorOption {
	return monitorOptionFunc(func(m *Monitor) error {
		if i <= 0 {
			i = DefaultProbeInterval
		}

		m.defaultProbeInterval = i
		return nil
	})
}

// WithSubsystems defines several subsystems for the monitor.
func WithSubsystems(defs ...Definition) MonitorOption {
	return monitorOptionFunc(func(m *Monitor) error {
		for _, d := range defs {
			if m.byName[d.Name] != nil {
				return fmt.Errorf("A subsystem with the name [%s] already exists", d.Name)
			}

			st := &subsystemTracker{
				lock:              &m.lock,
				unsafeUpdateState: m.unsafeUpdateState,
				definition:        d,
			}

			m.byName[d.Name] = st
			m.trackers = append(m.trackers, st)
		}

		return nil
	})
}

// NewMonitor constructs a health Monitor using the supplied
// set of options. The returned Monitor will not be running and
// must be started in order to receive Probe updates.
//
// The set of subsystems is fixed and immutable after construction.
// The initial value returned by the Monitor from the State method will
// be computed from the initial states of the subsystems.
// If no subsystems are configured in the options, the returned
// Monitor will always report StatusGood as its overall status.
func NewMonitor(opts ...MonitorOption) (*Monitor, error) {
	m := &Monitor{
		byName:               make(map[Name]*subsystemTracker),
		defaultProbeInterval: DefaultProbeInterval,
		now:                  time.Now,
		newTimer:             defaultNewTimer,
	}

	for _, o := range opts {
		if err := o.apply(m); err != nil {
			return nil, err
		}
	}

	m.subsystems = make([]Subsystem, len(m.trackers))

	// now that the options are applied, make a pass over the subsystems
	initialLastUpdate := m.now().UTC()
	for i, sst := range m.trackers {
		// pass the initialLastUpdate so all subsystem's get a consistent
		// starting timestamp.
		sst.initialize(m, initialLastUpdate, &m.subsystems[i])
	}

	m.unsafeUpdateState(initialLastUpdate)
	return m, nil
}
