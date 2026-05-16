// Package scaler implements a count-based Woodpecker agent scaler. Each Pool
// owns a slice of the workflow queue (selected by label match) and a target
// Nomad task group whose count is adjusted to satisfy demand.
package scaler

import (
	"fmt"
	"strings"
)

// Pool maps a queue selector to a Nomad task group plus capacity bounds.
type Pool struct {
	Name     string
	Group    string
	Selector LabelSelector
	Min      int
	Max      int
}

// LabelSelector matches a task by its labels. The zero value is a catch-all
// that owns any task no other selector claims (see Router).
type LabelSelector struct {
	Key   string
	Value string
}

// IsCatchAll reports whether this selector matches anything.
func (s LabelSelector) IsCatchAll() bool { return s.Key == "" }

// Matches reports whether the given labels satisfy the selector.
func (s LabelSelector) Matches(labels map[string]string) bool {
	if s.IsCatchAll() {
		return true
	}
	return labels[s.Key] == s.Value
}

// ParseSelector parses a "key=value" string. The empty string yields a
// catch-all selector.
func ParseSelector(s string) (LabelSelector, error) {
	if s == "" {
		return LabelSelector{}, nil
	}
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" || v == "" {
		return LabelSelector{}, fmt.Errorf("invalid selector %q: want key=value", s)
	}
	return LabelSelector{Key: k, Value: v}, nil
}
