// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProbeTestSuite struct {
	suite.Suite

	testCtx context.Context
}

func (suite *ProbeTestSuite) SetupSuite() {
	type contextKey struct{}
	suite.testCtx = context.WithValue(context.Background(), contextKey{}, "value")
}

func (suite *ProbeTestSuite) assertCtx(ctx context.Context) {
	suite.Same(suite.testCtx, ctx)
}

func (suite *ProbeTestSuite) assertProbe(p Probe, expectedStatus Status, expectedErr error) {
	suite.Require().NotNil(p)
	actualStatus, actualErr := p(suite.testCtx)
	suite.Equal(expectedStatus, actualStatus)
	suite.Equal(expectedErr, actualErr)
}

func (suite *ProbeTestSuite) testAsProbeReturnBool() {
	suite.Run("WithoutContext", func() {
		suite.Run("True", func() {
			var called bool
			pf := func() bool { called = true; return true }
			suite.assertProbe(AsProbe(pf), StatusGood, nil)
			suite.True(called)
		})

		suite.Run("False", func() {
			var called bool
			pf := func() bool { called = true; return false }
			suite.assertProbe(AsProbe(pf), StatusBad, nil)
			suite.True(called)
		})
	})

	suite.Run("WithContext", func() {
		suite.Run("True", func() {
			var called bool
			pf := func(ctx context.Context) bool { suite.assertCtx(ctx); called = true; return true }
			suite.assertProbe(AsProbe(pf), StatusGood, nil)
			suite.True(called)
		})

		suite.Run("False", func() {
			var called bool
			pf := func(ctx context.Context) bool { suite.assertCtx(ctx); called = true; return false }
			suite.assertProbe(AsProbe(pf), StatusBad, nil)
			suite.True(called)
		})
	})
}

func (suite *ProbeTestSuite) testAsProbeReturnError() {
	suite.Run("WithoutContext", func() {
		suite.Run("NoError", func() {
			var called bool
			pf := func() error { called = true; return nil }
			suite.assertProbe(AsProbe(pf), StatusGood, nil)
			suite.True(called)
		})
	})
}

func (suite *ProbeTestSuite) TestAsProbe() {
	suite.Run("ReturnBool", suite.testAsProbeReturnBool)
	suite.Run("ReturnError", suite.testAsProbeReturnError)
}

func TestProbe(t *testing.T) {
	suite.Run(t, new(ProbeTestSuite))
}
