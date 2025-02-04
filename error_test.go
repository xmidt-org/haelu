// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type boolError struct {
	msg     string
	success bool
}

func (be *boolError) Error() string {
	return be.msg
}

func (be *boolError) Status() bool {
	return be.success
}

type ErrorTestSuite struct {
	suite.Suite
}

func (suite *ErrorTestSuite) TestAddStatus() {
	testCases := []Status{
		StatusGood,
		StatusWarn,
		StatusBad,
	}

	for _, status := range testCases {
		wrappedErr := errors.New("wrapped error")
		err := AddStatus(wrappedErr, status)
		suite.Require().NotNil(err)
		suite.Require().Implements((*SelfStatuser)(nil), err)
		suite.Equal(wrappedErr.Error(), err.Error())
		suite.ErrorIs(err, wrappedErr)
		suite.Equal(status, err.(SelfStatuser).Status())
	}
}

func (suite *ErrorTestSuite) TestBooler() {
	suite.Run("True", func() {
		var err error = &boolError{
			msg:     "wrapped error",
			success: true,
		}

		suite.Equal(StatusGood, ErrorStatus(err))
	})
	suite.Run("False", func() {
		var err error = &boolError{
			msg:     "wrapped error",
			success: false,
		}

		suite.Equal(StatusBad, ErrorStatus(err))
	})
}

func (suite *ErrorTestSuite) TestErrorStatus() {
	testCases := []struct {
		name     string
		err      error
		expected Status
	}{
		{
			name:     "Nil",
			err:      nil,
			expected: StatusGood,
		},
		{
			name:     "Simple",
			err:      errors.New("expected"),
			expected: StatusBad,
		},
		{
			name: "TrueBooler",
			err: &boolError{
				msg:     "expected",
				success: true,
			},
			expected: StatusGood,
		},
		{
			name: "FalseBooler",
			err: &boolError{
				msg:     "expected",
				success: false,
			},
			expected: StatusBad,
		},
		{
			name: "GoodStatuser",
			err: &statusError{
				err:    errors.New("expected"),
				status: StatusGood,
			},
			expected: StatusGood,
		},
		{
			name: "WarnStatuser",
			err: &statusError{
				err:    errors.New("expected"),
				status: StatusWarn,
			},
			expected: StatusWarn,
		},
		{
			name: "BadStatuser",
			err: &statusError{
				err:    errors.New("expected"),
				status: StatusBad,
			},
			expected: StatusBad,
		},
	}

	for _, testCase := range testCases {
		suite.Run(testCase.name, func() {
			suite.Equal(
				testCase.expected,
				ErrorStatus(testCase.err),
			)
		})
	}
}

func TestError(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}
