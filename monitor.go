// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

// subsystemTracker holds all the information for tracking the state of
// a single subsystem.
type subsystemTracker struct {
	// lock is "inherited" from the containing monitor
	lock sync.Locker

	// newTimer is the timer strategy "inherited" from the containing monitor
	newTimer newTimer

	// unsafeUpdateStatus is the "inherited" non-atomic closure that updates monitor
	// status.
	unsafeUpdateStatus func(time.Time)

	// definition is the configuration used to create this subsystem
	definition Definition

	// current represents the current state of this monitor.  This is a pointer
	// into an element of the Monitor's subsystems.
	current *Subsystem
}

// startProbeTask ensures that a background goroutine is running
// to monitor the results from a Probe. If this subsystem has no Probe,
// this method does nothing.
//
// If this method starts a goroutine, it will stop with the supplied
// context is canceled.
func (ssm *subsystemTracker) startProbeTask(ctx context.Context) {
	if ssm.definition.Probe == nil {
		return
	}

	go func() {
		for {
			timeCh, stop := ssm.newTimer(ssm.definition.ProbeInterval)
			select {
			case <-ctx.Done():
				stop()
				return

			case <-timeCh:
				s, err := ssm.definition.Probe(ctx)
				ssm.Update(s, err)
			}
		}
	}()
}

// Update implements the Updater interface. This method updates this
// tracker's state under the monitor's lock. It then invokes the
// unsafeUpdateStatus closure to allow the monitor to update its
// overall status.
func (ssm *subsystemTracker) Update(s Status, err error) {
	defer ssm.lock.Unlock()
	ssm.lock.Lock()

	ssm.current.Status = s
	ssm.current.LastError = err
	ssm.current.LastUpdate = time.Now().UTC()

	ssm.unsafeUpdateStatus(ssm.current.LastUpdate)
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

	// newTimer is a factory for creating the timer channel and stop function.
	// if unset, defaultNewTimer is used.
	//
	// Tests can replace this function to control probe monitoring.
	newTimer newTimer

	byName     map[Name]*subsystemTracker
	trackers   []*subsystemTracker
	subsystems []Subsystem
	listeners  MonitorListeners

	lock sync.RWMutex

	// status is the overall health Status of this monitor.
	// this value is recomputed whenever a health update happens.
	status Status

	// lastUpdate is the timestamp of the last overall status change.
	// Start will always set this to the current time.
	lastUpdate time.Time

	// cancel is the cancellation function used to control any probe tasks
	cancel context.CancelFunc
}

// subsystemIter is an iter.Seq[Subsystem] that permits read-only
// access to the current subsystems.
func (m *Monitor) subsystemIter(f func(Subsystem) bool) {
	for _, s := range m.subsystems {
		if !f(s) {
			return
		}
	}
}

// unsafeUpdateStatus performs the following:
//
// (1) computes the (possibly) new overall status based on the current subystem states
// (2) dispatches events to configured listeners
//
// The timestamp of the update is supplied so that it's consistent with the timestamp
// of any individual subsystem updates.
//
// This method must be executed under the monitor lock or in a situation where no
// concurrent invocation is possible.
func (m *Monitor) unsafeUpdateStatus(timestamp time.Time) {
	m.lastUpdate = timestamp

	var (
		criticalStatus    Status
		nonCriticalStatus Status
	)

	for _, ssm := range m.trackers {
		switch {
		case ssm.definition.NonCritical && ssm.current.Status > nonCriticalStatus:
			nonCriticalStatus = ssm.current.Status

		case !ssm.definition.NonCritical && ssm.current.Status > criticalStatus:
			criticalStatus = ssm.current.Status
		}
	}

	switch {
	case criticalStatus != StatusGood:
		m.status = criticalStatus

	case nonCriticalStatus != StatusGood:
		m.status = StatusWarn

	default:
		m.status = StatusGood
	}

	m.listeners.OnMonitorEvent(
		MonitorEvent{
			Status:         m.status,
			LastUpdate:     m.lastUpdate,
			SubsystemCount: len(m.subsystems),
			Subsystems:     m.subsystemIter,
		},
	)
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

// Status returns the overall health status for this Monitor. If this Monitor has
// not been started, this method returns StatusGood. If this Monitor has been Shutdown
// and then restarted, it returns whatever the last computed status was before it
// was shutdown.
func (m *Monitor) Status() (current Status) {
	m.lock.RLock()
	current = m.status
	m.lock.RUnlock()
	return
}

// Start computes the initial, overall state based on the status of the subystems
// and then starts any background tasks to monitor subsystem Probes. A Monitor may
// receive updates from subsystems at any time, even before Start is called.
//
// This method is idempotent. If this Monitor has already been started, this method
// does nothing and returns ErrMonitorStarted.
//
// If a Monitor is Shutdown and then Started again, the previous states of all
// subsystems are retained.
func (m *Monitor) Start() error {
	defer m.lock.Unlock()
	m.lock.Lock()

	if m.cancel != nil {
		return ErrMonitorStarted
	}

	m.unsafeUpdateStatus(time.Now().UTC())
	var rootCtx context.Context
	rootCtx, m.cancel = context.WithCancel(context.Background())
	for _, ssm := range m.trackers {
		ssm.startProbeTask(rootCtx)
	}

	return nil
}

// Shutdown sets the overall status to StatusNotStarted and then ensures no
// background tasks for Probe monitoring are running. A Monitor may receive
// updates from subsystems at any time, even after Shutdown is called.
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

// WithSubsystem defines a single subsystem for health monitoring.
func WithSubsystem(d Definition) MonitorOption {
	return monitorOptionFunc(func(m *Monitor) error {
		if m.byName[d.Name] != nil {
			return fmt.Errorf("A subsystem with the name [%s] already exists", d.Name)
		}

		ssm := &subsystemTracker{
			lock:               &m.lock,
			unsafeUpdateStatus: m.unsafeUpdateStatus,
			definition:         d,
		}

		ssm.definition.Attributes = d.Attributes.Clone()
		m.byName[d.Name] = ssm
		m.trackers = append(m.trackers, ssm)
		return nil
	})
}

// WithListeners adds listeners to the Monitor.
func WithListeners(ls ...MonitorListener) MonitorOption {
	return monitorOptionFunc(func(m *Monitor) error {
		m.listeners = append(m.listeners, ls...)
		return nil
	})
}

// NewMonitor constructs a health Monitor using the supplied
// set of options. The returned Monitor will not be running and
// must be started in order to receive Probe updates.
//
// The set of subsystems is fixed and immutable after construction.
// If no subsystems are configured in the options, the returned
// Monitor will always report StatusGood as its overall status.
func NewMonitor(opts ...MonitorOption) (*Monitor, error) {
	m := &Monitor{
		byName:               make(map[Name]*subsystemTracker),
		defaultProbeInterval: DefaultProbeInterval,
		newTimer:             defaultNewTimer,
		status:               StatusGood,
	}

	for _, o := range opts {
		if err := o.apply(m); err != nil {
			return nil, err
		}
	}

	m.subsystems = make([]Subsystem, len(m.trackers))

	// now that the options are applied, make a pass over the subsystems
	initialLastUpdate := time.Now().UTC()
	for i, ssm := range m.trackers {
		ssm.newTimer = m.newTimer

		// take the initial state from the definition
		ssm.current = &m.subsystems[i]
		ssm.current.Status = ssm.definition.Status
		ssm.current.Attributes = ssm.definition.Attributes
		ssm.current.LastUpdate = initialLastUpdate

		// normalize the probe interval
		if ssm.definition.Probe == nil {
			ssm.definition.ProbeInterval = 0
		} else if ssm.definition.ProbeInterval <= 0 {
			ssm.definition.ProbeInterval = m.defaultProbeInterval
		}
	}

	return m, nil
}
