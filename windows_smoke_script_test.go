package main

import (
	"os"
	"strings"
	"testing"
)

func TestWindowsSmokeScriptExitsExplicitlyOnSuccess(t *testing.T) {
	data, err := os.ReadFile("scripts/windows-smoke.ps1")
	if err != nil {
		t.Fatalf("ReadFile(windows-smoke.ps1): %v", err)
	}

	script := string(data)
	if !strings.Contains(script, "exit 0") {
		t.Fatalf("windows smoke script should exit 0 explicitly after successful validation")
	}
}
