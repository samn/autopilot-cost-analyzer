package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	// Save and restore original values.
	origVersion, origCommit, origDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = origVersion, origCommit, origDate
	})

	version = "v1.2.3"
	commit = "abc1234"
	date = "2026-01-15T00:00:00Z"

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := buf.String()
	want := "gke-cost-analyzer v1.2.3 (commit: abc1234, built: 2026-01-15T00:00:00Z)\n"
	if got != want {
		t.Errorf("version output = %q, want %q", got, want)
	}
}
