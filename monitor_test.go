// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/chronon"
)

type MonitorTestSuite struct {
	suite.Suite

	// start is set to the start time of the (sub) test.  all timestamps
	// must be greater than or equal to this timestamp.
	start time.Time

	// clock is the fake clock used by all Monitors under test.
	clock *chronon.FakeClock
}

func (suite *MonitorTestSuite) initializeTime() {
	suite.start = time.Now()
	suite.clock = chronon.NewFakeClock(suite.start)
}

func (suite *MonitorTestSuite) SetupSuite() {
	suite.initializeTime()
}

func (suite *MonitorTestSuite) SetupTest() {
	suite.initializeTime()
}

func (suite *MonitorTestSuite) SetupSubTest() {
	suite.initializeTime()
}

// startUTC is a convenience for obtaining the start time of the
// current test as a UTC time.
func (suite *MonitorTestSuite) startUTC() time.Time {
	return suite.start.UTC()
}

// nowUTC is just a convenience to obtain the current UTC time
// of this test's clock.
func (suite *MonitorTestSuite) nowUTC() time.Time {
	return suite.clock.Now().UTC()
}

// names returns the Name of each subsystem.  useful for expected data.
func (suite *MonitorTestSuite) names(defs ...Definition) (ns []Name) {
	ns = make([]Name, len(defs))
	for i, d := range defs {
		ns[i] = d.Name
	}

	return
}

// newExpectedSubsystem produces the expected initial snapshot for a given definition.
// The returned Subsystem is what a Monitor should return initially for that definition.
// This will include the current value of the test clock.
func (suite *MonitorTestSuite) newExpectedSubsystem(d Definition) (s Subsystem) {
	s.Name = d.Name
	s.Status = d.Status
	s.Metadata = d.Metadata
	s.NonCritical = d.NonCritical
	s.LastUpdate = suite.startUTC()
	return
}

// newExpectedSubsystems creates one expected, initial snapshot for each definition,
// in the same order.
func (suite *MonitorTestSuite) newExpectedSubsystems(defs ...Definition) (ss []Subsystem) {
	if len(defs) > 0 {
		ss = make([]Subsystem, len(defs))
		for i := range len(defs) {
			ss[i] = suite.newExpectedSubsystem(defs[i])
		}
	}

	return
}

// newMonitor creates a new Monitor, asserts that construction worked correctly, and
// returns a FakeClock that can be used to control the Monitor's timer.
func (suite *MonitorTestSuite) newMonitor(o ...MonitorOption) *Monitor {
	o = append(o,
		monitorOptionFunc(func(m *Monitor) error {
			m.now = suite.clock.Now
			m.newTimer = fakeTimer(suite.clock)
			return nil
		}),
	)

	m, err := NewMonitor(o...)
	suite.Require().NoError(err)
	suite.Require().NotNil(m)
	return m
}

// assertState asserts that the given Monitor has the expected overall status, LastUpdate matches
// the test clock, and the subsystems match the given subsystems.
func (suite *MonitorTestSuite) assertState(m *Monitor, expected Status, subs ...Subsystem) {
	suite.Require().NotNil(m)
	state := m.State()
	suite.Equal(expected, state.Status)
	suite.Equal(suite.nowUTC(), state.LastUpdate)
	suite.Equal(len(subs), m.Len())
	suite.Equal(subs, state.Subsystems.ss)
}

// assertUpdater verifies that there is a subsystem with the give name and returns
// the Updater for that subsystem.
func (suite *MonitorTestSuite) assertUpdater(m *Monitor, name Name) Updater {
	u, err := m.Get(name)
	suite.Require().NoError(err)
	suite.Require().NotNil(u)
	return u
}

// assertUpdaters verifies several subsystem names via assertUpdater.
func (suite *MonitorTestSuite) assertUpdaters(m *Monitor, names ...Name) (us []Updater) {
	us = make([]Updater, len(names))
	for i := range len(names) {
		us[i] = suite.assertUpdater(m, names[i])
	}

	return
}

// assertStart checks that the Monitor can be started and that Start
// is idempotent.
func (suite *MonitorTestSuite) assertStart(m *Monitor) {
	suite.NoError(m.Start())
	suite.ErrorIs(m.Start(), ErrMonitorStarted) // idempotent
}

// assertShutdown checks that the Monitor can be shutdown and that Shutdown
// is idempotent.
func (suite *MonitorTestSuite) assertShutdown(m *Monitor) {
	suite.NoError(m.Shutdown())
	suite.ErrorIs(m.Shutdown(), ErrMonitorShutdown) // idempotent
}

func (suite *MonitorTestSuite) TestInitialStates() {
	testCases := []struct {
		name        string
		definitions []Definition
		expected    Status
	}{
		{
			name:        "NoSubsystems",
			definitions: nil,
			expected:    StatusGood,
		},
		{
			name: "OneGoodCritical",
			definitions: []Definition{
				{Name: "initial"},
			},
			expected: StatusGood,
		},
		{
			name: "OneWarnCritical",
			definitions: []Definition{
				{Name: "initial", Status: StatusWarn},
			},
			expected: StatusWarn,
		},
		{
			name: "OneBadCritical",
			definitions: []Definition{
				{Name: "initial", Status: StatusBad},
			},
			expected: StatusBad,
		},
	}

	for _, testCase := range testCases {
		suite.Run(testCase.name, func() {
			m := suite.newMonitor(WithSubsystems(testCase.definitions...))
			suite.assertState(
				m,
				testCase.expected,
				suite.newExpectedSubsystems(testCase.definitions...)...,
			)

			suite.assertUpdaters(m, suite.names(testCase.definitions...)...)

			suite.clock.Add(time.Second)
			suite.assertStart(m)
			suite.assertState(
				m,
				testCase.expected,
				suite.newExpectedSubsystems(testCase.definitions...)...,
			)

			suite.assertUpdaters(m, suite.names(testCase.definitions...)...)
		})
	}
}

func TestMonitor(t *testing.T) {
	suite.Run(t, new(MonitorTestSuite))
}
