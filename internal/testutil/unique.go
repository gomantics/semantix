package testutil

import (
	"fmt"
	"sync/atomic"
)

var counter atomic.Uint64

// UniqueID returns a unique string safe for use across parallel tests.
// Each call returns a different value within a test binary invocation.
func UniqueID() string {
	return fmt.Sprintf("%d", counter.Add(1))
}
