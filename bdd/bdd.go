// Package bdd holds the given-when-then metadata every test starts with (see "Testing" in AGENTS.md).
// The metadata is plain English and describes the behavior, so agents can read what a test protects
// without reading its body.
package bdd

import (
	"strings"
	"testing"
)

type BDD struct {
	Given string
	When  string
	Then  string
}

// Behavior fails the test when any of Given, When or Then is empty.
func Behavior(t testing.TB, behavior BDD) {
	t.Helper()

	missingFields := []string{}
	if strings.TrimSpace(behavior.Given) == "" {
		missingFields = append(missingFields, "Given")
	}
	if strings.TrimSpace(behavior.When) == "" {
		missingFields = append(missingFields, "When")
	}
	if strings.TrimSpace(behavior.Then) == "" {
		missingFields = append(missingFields, "Then")
	}
	if len(missingFields) > 0 {
		t.Fatalf("BDD behavior metadata is missing %s", strings.Join(missingFields, ", "))
	}
}
