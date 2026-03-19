package main

import (
	"os"
	"strings"
	"testing"
)

func TestGoReleaserConfigDoesNotReferenceMissingHomebrewTap(t *testing.T) {
	content, err := os.ReadFile(".goreleaser.yaml")
	if err != nil {
		t.Fatalf("read goreleaser config: %v", err)
	}

	config := string(content)

	if strings.Contains(config, "homebrew-tap") {
		t.Fatal("goreleaser config still references missing homebrew tap repository")
	}

	if strings.Contains(config, "\nbrews:") {
		t.Fatal("goreleaser config still enables Homebrew publishing")
	}
}
