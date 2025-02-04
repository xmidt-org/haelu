// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StatusTestSuite struct {
	suite.Suite
}

func (suite *StatusTestSuite) TestDistinctStrings() {
	// we don't care what the string values are, just that they're distinct
	m := make(map[string]bool)
	m[StatusGood.String()] = true
	m[StatusWarn.String()] = true
	m[StatusBad.String()] = true
	suite.Len(m, 3)
}

func TestStatus(t *testing.T) {
	suite.Run(t, new(StatusTestSuite))
}
