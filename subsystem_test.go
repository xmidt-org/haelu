// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SubsystemTestSuite struct {
	suite.Suite
}

func (suite *SubsystemTestSuite) testAsSubsystemsEmpty() {
	s := AsSubsystems()
	suite.Zero(s.Len())
	suite.Panics(func() {
		s.Get(0)
	})

	var called bool
	for range s.All() {
		called = true
	}

	suite.False(called)
}

func (suite *SubsystemTestSuite) testAsSubsystemsNotEmpty() {
	original := []Subsystem{
		{
			Name: "first",
			Metadata: Values(
				"test", "value",
			),
		},
		{
			Name:        "second",
			NonCritical: true,
		},
	}

	s := AsSubsystems(original...)
	suite.Equal(2, s.Len())
	suite.Equal(original[0], s.Get(0))
	suite.Equal(original[1], s.Get(1))
	suite.Panics(func() {
		s.Get(2)
	})

	var called bool
	i := 0
	for sub := range s.All() {
		suite.Equal(original[i], sub)
		i++
		called = true
	}

	suite.True(called)

	var count int
	for range s.All() {
		count++
		break
	}

	suite.Equal(1, count, "All needs to honor early return")
}

func (suite *SubsystemTestSuite) testAsSubsystemsMarshalJSON() {
	original := []Subsystem{
		{
			Name: "first",
			Metadata: Values(
				"test", "value",
			),
		},
		{
			Name:        "second",
			NonCritical: true,
		},
	}

	expected, err := json.Marshal(original)
	suite.Require().NoError(err)

	actual, err := AsSubsystems(original...).MarshalJSON()
	suite.Require().NoError(err)
	suite.JSONEq(string(expected), string(actual))
}

func (suite *SubsystemTestSuite) TestAsSubsystems() {
	suite.Run("Empty", suite.testAsSubsystemsEmpty)
	suite.Run("NotEmpty", suite.testAsSubsystemsNotEmpty)
	suite.Run("MarshalJSON", suite.testAsSubsystemsMarshalJSON)
}

func TestSubsystem(t *testing.T) {
	suite.Run(t, new(SubsystemTestSuite))
}
