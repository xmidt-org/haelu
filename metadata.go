// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package haelu

import (
	"encoding/json"
	"fmt"
	"iter"
)

// Metadata represents an immutable set of name/value pairs. Metadata may be
// associated with subsystems to expose client-specific metadata.
//
// The zero value of this type is a usable, empty set.
type Metadata struct {
	m map[string]any
}

// Len returns the number of name/value pairs in this set.
func (m Metadata) Len() int {
	return len(m.m)
}

// Get returns the value for a given name, if one exists.
func (m Metadata) Get(name string) (value any, exists bool) {
	value, exists = m.m[name]
	return
}

// All provides iteration over each name/value pair in this set.
func (m Metadata) All() iter.Seq2[string, any] {
	return func(f func(string, any) bool) {
		for n, v := range m.m {
			if !f(n, v) {
				return
			}
		}
	}
}

// String returns a string representation of this Metadata set.
func (m Metadata) String() string {
	return fmt.Sprintf("%s", m.m)
}

// MarshalJSON writes this Metadata as a JSON object.
func (m Metadata) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.m)
}

// Map returns an immutable Metadata with the same name/value pairs
// as the src map. The returned Attributes is a shallow copy of the src.
//
// Values will be of type any in the returned Metadata. This function is
// genericized to make it easier to turn any map with string keys into
// Metadata.
func Map[T any](src map[string]T) Metadata {
	m := Metadata{
		m: make(map[string]any, len(src)),
	}

	for n, v := range src {
		m.m[n] = v
	}

	return m
}

// toName just converts an arbitrary value into a string.
func toName(v any) string {
	if n, ok := v.(string); ok {
		return n
	} else if s, ok := v.(fmt.Stringer); ok {
		return s.String()
	}

	return fmt.Sprintf("%v", v)
}

// Values returns an immutable Metadata given a sequence of values.
// The values must occur in a series of pairs, e.g. name1, value1, name2, value2, etc.
// Thus, even index elements are names and odd index elements are the corresponding
// values. If there is an odd number of elements passed to this function, the
// last name is mapped to untyped nil in the returned attributes.
//
// If a name is not a string and does not implement fmt.Stringer, fmt.Sprintf
// is used to convert it into a string to use as a Metadata key.
func Values(v ...any) Metadata {
	m := Metadata{
		m: make(map[string]any, len(v)/2),
	}

	i, j := 0, 1
	for ; j < len(v); i, j = i+2, j+2 {
		m.m[toName(v[i])] = v[j]
	}

	if i < len(v) {
		m.m[toName(v[i])] = nil
	}

	return m
}
