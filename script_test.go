package imsg

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// hostileStrings would break out of an AppleScript string literal, or change
// its value, if escaping were wrong.
var hostileStrings = []string{
	"",
	"plain",
	`say "hi"`,
	`back\slash`,
	`trailing\`,
	`" & (do shell script "echo pwned") & "`,
	"line1\nline2\r\nline3\ttabbed",
	"curly “quotes” and «chevrons»",
	"emoji 🎉 and ünïcödé",
}

func requireDarwinTool(t *testing.T, name string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("requires macOS")
	}
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available: %v", name, err)
	}
}

// TestEscapeAppleScriptStringRoundTrip has the real AppleScript compiler
// evaluate each escaped literal, proving the value survives unchanged.
func TestEscapeAppleScriptStringRoundTrip(t *testing.T) {
	t.Parallel()
	requireDarwinTool(t, "osascript")

	for _, value := range hostileStrings {
		script := `return "` + escapeAppleScriptString(value) + `"`
		out, err := exec.CommandContext(t.Context(), "osascript", "-e", script).CombinedOutput() //nolint:gosec // test-only; exercises the production escaping
		if err != nil {
			t.Errorf("osascript failed for %q: %v: %s", value, err, out)
			continue
		}
		if got := string(out); got != value+"\n" {
			t.Errorf("round trip of %q returned %q", value, got)
		}
	}
}

// TestBuildScriptsCompile checks generated scripts against the Messages
// scripting dictionary without running them or launching Messages.
func TestBuildScriptsCompile(t *testing.T) {
	t.Parallel()
	requireDarwinTool(t, "osacompile")

	attachments := []attachmentInfo{
		{Path: `/tmp/with "quotes" and \slashes.png`, DelaySecond: 2},
		{Path: "/tmp/no-delay.mov"},
	}
	text := `hello "there"` + "\nsecond line"

	scripts := map[string]string{
		"send":           buildSendScript("+12025550123", text, attachments),
		"send text only": buildSendScript("someone@example.com", text, nil),
		"chat":           buildSendChatScript("any;+;chat123", text, attachments),
		"chat file only": buildSendChatScript("any;+;chat123", "", attachments),
	}
	for name, script := range scripts {
		out := filepath.Join(t.TempDir(), "out.scpt")
		cmd := exec.CommandContext(t.Context(), "osacompile", "-o", out, "-e", script) //nolint:gosec // test-only; compiles generated script
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s: osacompile failed: %v: %s\n%s", name, err, output, script)
		}
	}
}
