// Package testutil provides shared test infrastructure for integration tests.
// Use testutil.Main in TestMain functions to compose infrastructure options.
package testutil

import (
	"flag"
	"testing"
	"time"
)

// AcceptChanges is a flag that, when set, auto-approves all snapshot diffs.
var AcceptChanges bool

func init() {
	flag.BoolVar(&AcceptChanges, "accept-changes", false, "automatically accept approval test snapshots")
}

// Teardown is a cleanup function returned by an Option.
type Teardown func()

// Option sets up a piece of test infrastructure and returns its teardown.
type Option func() Teardown

// M wraps *testing.M and manages ordered infrastructure setup and teardown.
type M struct {
	m         *testing.M
	teardowns []Teardown
}

// Main creates a new M with the given options, running each option's setup
// immediately. Teardowns are run in reverse order after m.Run().
func Main(m *testing.M, opts ...Option) *M {
	// Force UTC so timestamp assertions are deterministic.
	time.Local = time.UTC

	hm := &M{m: m}
	for _, opt := range opts {
		td := opt()
		hm.teardowns = append(hm.teardowns, td)
	}
	return hm
}

// Run executes the test suite, then runs all teardowns in reverse order.
func (m *M) Run() int {
	code := m.m.Run()

	for i := len(m.teardowns) - 1; i >= 0; i-- {
		m.teardowns[i]()
	}

	return code
}
