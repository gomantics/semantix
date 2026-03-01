// Package testflags registers shared test flags.
package testflags

import "flag"

// AcceptChanges is true when -accept-changes is passed to go test.
var AcceptChanges bool

func init() {
	flag.BoolVar(&AcceptChanges, "accept-changes", false, "automatically accept approval test snapshots")
}
