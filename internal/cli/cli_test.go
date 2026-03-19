package cli

import (
	"reflect"
	"testing"
)

func TestParseInvocation_DebugBeforeCommand(t *testing.T) {
	cmd, subArgs, debug, ok := parseInvocation([]string{"crontui", "--debug", "completion", "bash"})
	if !ok {
		t.Fatal("parseInvocation should handle a leading --debug flag")
	}
	if !debug {
		t.Fatal("parseInvocation should report debug=true when --debug is present")
	}
	if cmd != "completion" {
		t.Fatalf("cmd = %q, want %q", cmd, "completion")
	}
	if !reflect.DeepEqual(subArgs, []string{"bash"}) {
		t.Fatalf("subArgs = %#v, want %#v", subArgs, []string{"bash"})
	}
}

func TestParsePreviewArgs_RejectsNonPositiveCount(t *testing.T) {
	for _, args := range [][]string{
		{"0 * * * *", "0"},
		{"0 * * * *", "-1"},
	} {
		if _, _, err := parsePreviewArgs(args); err == nil {
			t.Fatalf("parsePreviewArgs(%#v) should reject non-positive counts", args)
		}
	}
}
