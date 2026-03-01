package testutil

import (
	"encoding/json"
	"io"

	approvals "github.com/approvals/go-approval-tests"
	"github.com/approvals/go-approval-tests/reporters"
	"github.com/gomantics/semantix/internal/testflags"
)

// WithApprovals returns an Option that configures the go-approval-tests reporter
// and snapshot folder. When the -accept-changes flag is set, all diffs are
// auto-approved.
func WithApprovals() Option {
	return func() Teardown {
		return withApprovals()
	}
}

func withApprovals() Teardown {
	approvals.UseFolder("testdata")

	var closer io.Closer
	if testflags.AcceptChanges {
		closer = approvals.UseFrontLoadedReporter(reporters.NewReporterThatAutomaticallyApproves())
	} else {
		closer = approvals.UseReporter(reporters.NewQuietReporter())
	}

	return func() {
		if closer != nil {
			closer.Close()
		}
	}
}

// ScrubField replaces the value of a key in a decoded JSON map with "[SCRUBBED]".
// This is used before snapshot assertions to mask non-deterministic fields like
// IDs and timestamps.
func ScrubField(data map[string]any, field string) {
	if _, ok := data[field]; ok {
		data[field] = "[SCRUBBED]"
	}
}

// ScrubFields scrubs multiple fields from a decoded JSON map.
func ScrubFields(data map[string]any, fields ...string) {
	for _, f := range fields {
		ScrubField(data, f)
	}
}

// MustMarshalJSON marshals v to a JSON string, panicking on error. Useful for
// building approval test inputs inline.
func MustMarshalJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}
