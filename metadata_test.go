// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type stringerName struct{}

func (stringerName) String() string { return "stringer" }

type MetadataTestSuite struct {
	suite.Suite
}

func (suite *MetadataTestSuite) assertValue(src Metadata, name string, expected any) {
	actual, ok := src.Get(name)
	suite.True(ok)
	suite.Equal(expected, actual)
}

func (suite *MetadataTestSuite) TestMap() {
	suite.Run("StringValue", func() {
		data := map[string]string{
			"url":        "https://remote.system.net:8080/foo/bar",
			"datacenter": "abcd",
		}

		m := Map(data)
		_, exists := m.Get("nosuch")
		suite.False(exists)

		suite.Equal(2, m.Len())
		suite.assertValue(m, "url", data["url"])
		suite.assertValue(m, "datacenter", data["datacenter"])
		for n, v := range m.All() {
			suite.Equalf(data[n], v, "name not in original data: %s", n)
		}
	})

	suite.Run("AnyValue", func() {
		data := map[string]any{
			"host": "hostname.net",
			"port": 8080,
		}

		m := Map(data)
		_, exists := m.Get("nosuch")
		suite.False(exists)

		suite.Equal(2, m.Len())
		suite.assertValue(m, "host", data["host"])
		suite.assertValue(m, "port", data["port"])
		for n, v := range m.All() {
			suite.Equalf(data[n], v, "name not in original data: %s", n)
		}
	})
}

func (suite *MetadataTestSuite) TestValues() {
	suite.Run("EvenNumber", func() {
		m := Values(
			"url", "https://service.foobar.net/something",
			"version", 1.2,
			123, "a number",
			stringerName{}, "stringer value",
		)

		_, exists := m.Get("nosuch")
		suite.False(exists)

		suite.Equal(4, m.Len())
		suite.assertValue(m, "url", "https://service.foobar.net/something")
		suite.assertValue(m, "version", 1.2)
		suite.assertValue(m, "123", "a number")
		suite.assertValue(m, "stringer", "stringer value")
		for n, v := range m.All() {
			switch n {
			case "url":
				suite.Equal(v, "https://service.foobar.net/something")

			case "version":
				suite.Equal(v, 1.2)

			case "123":
				suite.Equal(v, "a number")

			case "stringer":
				suite.Equal(v, "stringer value")

			default:
				suite.Failf("key not in original values", "key: %s", n)
			}
		}
	})

	suite.Run("OddNumber", func() {
		m := Values(
			"url", "https://service.foobar.net/something",
			"version", 1.2,
			"dangling",
		)

		_, exists := m.Get("nosuch")
		suite.False(exists)

		suite.Equal(3, m.Len())
		suite.assertValue(m, "url", "https://service.foobar.net/something")
		suite.assertValue(m, "version", 1.2)
		suite.assertValue(m, "dangling", nil)
		for n, v := range m.All() {
			switch n {
			case "url":
				suite.Equal(v, "https://service.foobar.net/something")

			case "version":
				suite.Equal(v, 1.2)

			case "dangling":
				suite.Nil(v)

			default:
				suite.Failf("key not in original values", "key: %s", n)
			}
		}
	})
}

func (suite *MetadataTestSuite) TestString() {
	m := Values(
		"foo", "bar",
	)

	suite.Equal(
		fmt.Sprintf("%s", map[string]any{"foo": "bar"}),
		m.String(),
	)
}

func (suite *MetadataTestSuite) TestMarshalJSON() {
	m := Values(
		"foo", "bar",
	)

	data, err := json.Marshal(m)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data)
	suite.JSONEq(`{"foo": "bar"}`, string(data))
}

func TestMetadata(t *testing.T) {
	suite.Run(t, new(MetadataTestSuite))
}
