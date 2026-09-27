package imsg

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExpandHome(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}

	cases := map[string]string{
		"~":         home,
		"~/a/b.png": filepath.Join(home, "a", "b.png"),
		"/abs/path": "/abs/path",
		"rel/path":  "rel/path",
	}
	for in, want := range cases {
		got, err := expandHome(in)
		if err != nil || got != want {
			t.Errorf("expandHome(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	if _, err := expandHome("~other/x"); !errors.Is(err, ErrUnsupportedTildePath) {
		t.Errorf("expandHome(~other/x) error = %v; want ErrUnsupportedTildePath", err)
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func tempCopies(t *testing.T, home string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(home, "Pictures", ".imsg_temp_*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestNormalizeAttachmentsErrors(t *testing.T) {
	// A fake HOME keeps attachment copies out of the real ~/Pictures.
	home := t.TempDir()
	t.Setenv("HOME", home)
	outside := filepath.Join(t.TempDir(), "outside.png")
	writeFile(t, outside)

	cases := []struct {
		paths []string
		want  error
	}{
		{[]string{"  "}, ErrAttachmentPathEmpty},
		{[]string{home}, ErrAttachmentIsDirectory},
		{[]string{filepath.Join(home, "missing.png")}, fs.ErrNotExist},
		// The first attachment is copied before the second fails.
		{[]string{outside, filepath.Join(home, "missing.png")}, fs.ErrNotExist},
	}
	for _, tc := range cases {
		if _, _, err := normalizeAttachments(tc.paths); !errors.Is(err, tc.want) {
			t.Errorf("normalizeAttachments(%q) error = %v; want %v", tc.paths, err, tc.want)
		}
	}

	if left := tempCopies(t, home); len(left) != 0 {
		t.Errorf("temp copies left behind after failure: %v", left)
	}
}

func TestNormalizeAttachmentsUsesSandboxFilesInPlace(t *testing.T) {
	// A fake HOME keeps attachment copies out of the real ~/Pictures.
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "Documents", "report.pdf")
	writeFile(t, path)

	attachments, cleanups, err := normalizeAttachments([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	if len(attachments) != 1 || attachments[0].Path != path || len(cleanups) != 0 {
		t.Errorf("got %+v with %d cleanups; want %s used in place", attachments, len(cleanups), path)
	}
}

// fakeOsascript writes a stand-in osascript to a new directory, for use as
// PATH. It saves the script it receives (its second argument), then runs body.
// It returns the directory and the path the script is saved to.
func fakeOsascript(t *testing.T, body string) (dir, saved string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("requires macOS")
	}
	dir = t.TempDir()
	saved = filepath.Join(dir, "script.txt")
	contents := "#!/bin/sh\nprintf '%s' \"$2\" > '" + saved + "'\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "osascript"), []byte(contents), 0o700); err != nil { //nolint:gosec // test fake must be executable
		t.Fatal(err)
	}
	return dir, saved
}

func TestSendValidationNeverRunsScript(t *testing.T) {
	bin, saved := fakeOsascript(t, "exit 0")
	t.Setenv("PATH", bin)
	client := NewClient()

	checks := []struct {
		err  error
		want error
	}{
		{client.Send("", Message{Text: "hi"}), ErrMissingRecipient},
		{client.Send("+12025550123", Message{Text: " \n\t"}), ErrEmptyMessage},
		{client.Send("+12025550123", Message{Text: "hi", Attachments: []string{""}}), ErrAttachmentPathEmpty},
		{client.SendChatID(" ", Message{Text: "hi"}), ErrMissingChatID},
		{client.SendChatID("any;+;chat123", Message{}), ErrEmptyMessage},
	}
	for i, c := range checks {
		if !errors.Is(c.err, c.want) {
			t.Errorf("check %d: error = %v; want %v", i, c.err, c.want)
		}
	}

	if _, err := os.Stat(saved); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("osascript was run for an invalid message")
	}
}

func TestSendCopiesAttachmentAndCleansUp(t *testing.T) {
	bin, saved := fakeOsascript(t, "exit 0")
	t.Setenv("PATH", bin)
	// A fake HOME keeps attachment copies out of the real ~/Pictures.
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(t.TempDir(), "Holiday Photo.png")
	writeFile(t, path)

	if err := NewClient().Send("+12025550123", Message{Text: "look", Attachments: []string{path}}); err != nil {
		t.Fatal(err)
	}

	script, err := os.ReadFile(saved) //nolint:gosec // path is from t.TempDir
	if err != nil {
		t.Fatal(err)
	}
	// The recipient sees the basename, so the copy must keep it.
	copyPrefix := filepath.Join(home, "Pictures", ".imsg_temp_")
	for _, want := range []string{`buddy "+12025550123"`, `send POSIX file "` + copyPrefix, `/Holiday Photo.png" to targetBuddy`, `send "look"`} {
		if !strings.Contains(string(script), want) {
			t.Errorf("script missing %q:\n%s", want, script)
		}
	}

	if left := tempCopies(t, home); len(left) != 0 {
		t.Errorf("temp copies left behind after send: %v", left)
	}
}

func TestSendReportsScriptFailures(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{`echo "execution error: Messages got an error: Can't get buddy id \"x\". (-1728)" >&2; exit 1`, "recipient not found"},
		{`echo "boom" >&2; exit 1`, "osascript failed: exit status 1: boom"},
	}
	for _, tc := range cases {
		bin, _ := fakeOsascript(t, tc.body)
		t.Setenv("PATH", bin)
		err := NewClient().SendText("+12025550123", "hi")
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("error = %v; want it to contain %q", err, tc.want)
		}
	}

	bin, _ := fakeOsascript(t, "exec /bin/sleep 5")
	t.Setenv("PATH", bin)
	err := NewClient(WithTimeout(100*time.Millisecond)).SendText("+12025550123", "hi")
	if !errors.Is(err, ErrScriptTimeout) {
		t.Errorf("error = %v; want ErrScriptTimeout", err)
	}
}
