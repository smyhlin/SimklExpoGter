//go:build !cli

package main

import "testing"

func TestShouldRunCLI(t *testing.T) {
	if shouldRunCLI([]string{"SimklExpoGter.exe"}) {
		t.Fatal("expected GUI mode when no subcommand is provided")
	}

	if !shouldRunCLI([]string{"SimklExpoGter.exe", "run"}) {
		t.Fatal("expected CLI mode when a known subcommand is provided")
	}

	if shouldRunCLI([]string{"SimklExpoGter.exe", "unknown"}) {
		t.Fatal("expected GUI mode for unknown arguments")
	}
}
