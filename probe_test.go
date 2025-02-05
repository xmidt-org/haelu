// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProbeTestSuite struct {
	suite.Suite

	called  bool
	testCtx context.Context
}

func (suite *ProbeTestSuite) SetupSuite() {
	type contextKey struct{}
	suite.testCtx = context.WithValue(context.Background(), contextKey{}, "value")
}

func (suite *ProbeTestSuite) SetupTest() {
	suite.called = false
}

func (suite *ProbeTestSuite) SetupSubTest() {
	suite.called = false
}

func (suite *ProbeTestSuite) assertCtx(ctx context.Context) {
	suite.Same(suite.testCtx, ctx)
}

func (suite *ProbeTestSuite) assertProbe(p Probe, expectedStatus Status, expectedErr error) {
	suite.Require().NotNil(p)
	actualStatus, actualErr := p(suite.testCtx)
	suite.Equal(expectedStatus, actualStatus)
	suite.ErrorIs(expectedErr, actualErr)
	suite.True(suite.called)
}

// testAsProbeReturnBool verifies func() bool and func(context.Context) bool
func (suite *ProbeTestSuite) testAsProbeReturnBool() {
	suite.Run("WithoutContext", func() {
		suite.Run("True", func() {
			pf := func() bool { suite.called = true; return true }
			suite.assertProbe(AsProbe(pf), StatusGood, nil)
		})

		suite.Run("False", func() {
			pf := func() bool { suite.called = true; return false }
			suite.assertProbe(AsProbe(pf), StatusBad, nil)
		})
	})

	suite.Run("WithContext", func() {
		suite.Run("True", func() {
			pf := func(ctx context.Context) bool { suite.assertCtx(ctx); suite.called = true; return true }
			suite.assertProbe(AsProbe(pf), StatusGood, nil)
		})

		suite.Run("False", func() {
			pf := func(ctx context.Context) bool { suite.assertCtx(ctx); suite.called = true; return false }
			suite.assertProbe(AsProbe(pf), StatusBad, nil)
		})
	})
}

// testAsProbeReturnError verifies func() error and func(context.Context) error
func (suite *ProbeTestSuite) testAsProbeReturnError() {
	suite.Run("WithoutContext", func() {
		for _, testCase := range errorStatusTestCases {
			suite.Run(testCase.name, func() {
				pf := func() error { suite.called = true; return testCase.err }
				suite.assertProbe(AsProbe(pf), testCase.expected, testCase.err)
			})
		}
	})

	suite.Run("WithContext", func() {
		for _, testCase := range errorStatusTestCases {
			suite.Run(testCase.name, func() {
				pf := func(ctx context.Context) error { suite.assertCtx(ctx); suite.called = true; return testCase.err }
				suite.assertProbe(AsProbe(pf), testCase.expected, testCase.err)
			})
		}
	})
}

// testAsProbeReturnStatus verifies func() Status and func(context.Context) Status
func (suite *ProbeTestSuite) testAsProbeReturnStatus() {
	testCases := []Status{
		StatusGood,
		StatusWarn,
		StatusBad,
	}

	suite.Run("WithoutContext", func() {
		for _, testCase := range testCases {
			suite.Run(testCase.String(), func() {
				pf := func() Status { suite.called = true; return testCase }
				suite.assertProbe(AsProbe(pf), testCase, nil)
			})
		}
	})

	suite.Run("WithContext", func() {
		for _, testCase := range testCases {
			suite.Run(testCase.String(), func() {
				pf := func(ctx context.Context) Status { suite.assertCtx(ctx); suite.called = true; return testCase }
				suite.assertProbe(AsProbe(pf), testCase, nil)
			})
		}
	})
}

// testAsProbeReturnStatusError verifies func() (Status, error) and func(context.Context) (Status, error)
// there's no error or Status translation in the production code, so these tests are
// much simpler.
func (suite *ProbeTestSuite) testAsProbeReturnStatusError() {
	suite.Run("WithoutContext", func() {
		err := errors.New("expected")
		pf := func() (Status, error) { suite.called = true; return StatusWarn, err }
		suite.assertProbe(AsProbe(pf), StatusWarn, err)
	})

	suite.Run("WithContext", func() {
		err := errors.New("expected")
		pf := func(ctx context.Context) (Status, error) {
			suite.assertCtx(ctx)
			suite.called = true
			return StatusWarn, err
		}

		suite.assertProbe(AsProbe(pf), StatusWarn, err)
	})
}

func (suite *ProbeTestSuite) TestAsProbe() {
	suite.Run("ReturnBool", suite.testAsProbeReturnBool)
	suite.Run("ReturnError", suite.testAsProbeReturnError)
	suite.Run("ReturnStatus", suite.testAsProbeReturnStatus)
	suite.Run("ReturnStatusError", suite.testAsProbeReturnStatusError)
}

func TestProbe(t *testing.T) {
	suite.Run(t, new(ProbeTestSuite))
}
