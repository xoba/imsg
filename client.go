package imsg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultScriptTimeout = 30 * time.Second

// Message is the content to send to a recipient.
type Message struct {
	Text        string
	Attachments []string
}

// Client sends iMessage content via AppleScript.
type Client struct {
	ScriptTimeout time.Duration
	Debug         bool
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the AppleScript execution timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.ScriptTimeout = timeout
		}
	}
}

// WithDebug enables debug logging.
func WithDebug(enabled bool) Option {
	return func(c *Client) {
		c.Debug = enabled
	}
}

// NewClient creates a new Client with optional settings.
func NewClient(opts ...Option) *Client {
	client := &Client{ScriptTimeout: defaultScriptTimeout}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// DefaultClient is used by package-level convenience functions.
var DefaultClient = NewClient()

// Send sends a message using the default client.
func Send(to string, msg Message) error {
	return DefaultClient.Send(to, msg)
}

// SendText sends a text-only iMessage using the default client.
func SendText(to, text string) error {
	return DefaultClient.SendText(to, text)
}

// SendFile sends a single attachment using the default client.
func SendFile(to, path string) error {
	return DefaultClient.SendFile(to, path)
}

// SendFiles sends multiple attachments with optional text using the default client.
func SendFiles(to string, paths []string, text string) error {
	return DefaultClient.SendFiles(to, paths, text)
}

// Send sends a message to a recipient.
func (c *Client) Send(to string, msg Message) error {
	if runtime.GOOS != "darwin" {
		return ErrUnsupportedOS
	}

	if strings.TrimSpace(to) == "" {
		return ErrMissingRecipient
	}

	trimmedText := strings.TrimSpace(msg.Text)

	attachments, cleanups, err := normalizeAttachments(msg.Attachments)
	if err != nil {
		return err
	}
	defer runCleanups(cleanups, c.Debug)

	if trimmedText == "" && len(attachments) == 0 {
		return ErrEmptyMessage
	}

	script := buildSendScript(to, msg.Text, attachments)
	return c.execAppleScript(script)
}

// SendText sends a text-only iMessage.
func (c *Client) SendText(to, text string) error {
	return c.Send(to, Message{Text: text})
}

// SendFile sends a single attachment.
func (c *Client) SendFile(to, path string) error {
	return c.Send(to, Message{Attachments: []string{path}})
}

// SendFiles sends multiple attachments with optional text.
func (c *Client) SendFiles(to string, paths []string, text string) error {
	return c.Send(to, Message{Text: text, Attachments: paths})
}

func (c *Client) execAppleScript(script string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.ScriptTimeout)
	defer cancel()

	if c.Debug {
		fmt.Fprintf(os.Stderr, "imsg: executing AppleScript:\n%s\n", script)
	}

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("imsg: osascript timeout after %s", c.ScriptTimeout)
	}

	if err != nil {
		if strings.Contains(trimmed, "Can't get buddy") {
			return fmt.Errorf("imsg: recipient not found: %w", err)
		}
		if trimmed != "" {
			return fmt.Errorf("imsg: osascript failed: %w: %s", err, trimmed)
		}
		return fmt.Errorf("imsg: osascript failed: %w", err)
	}

	if c.Debug && trimmed != "" {
		fmt.Fprintf(os.Stderr, "imsg: osascript output: %s\n", trimmed)
	}

	return nil
}

func normalizeAttachments(paths []string) ([]attachmentInfo, []func() error, error) {
	if len(paths) == 0 {
		return nil, nil, nil
	}

	attachments := make([]attachmentInfo, 0, len(paths))
	cleanups := make([]func() error, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			return nil, nil, fmt.Errorf("imsg: attachment path is empty")
		}

		expanded, err := expandHome(trimmed)
		if err != nil {
			runCleanups(cleanups, false)
			return nil, nil, err
		}

		absolute, err := filepath.Abs(expanded)
		if err != nil {
			runCleanups(cleanups, false)
			return nil, nil, fmt.Errorf("imsg: resolve attachment %q: %w", trimmed, err)
		}

		info, err := os.Stat(absolute)
		if err != nil {
			runCleanups(cleanups, false)
			return nil, nil, fmt.Errorf("imsg: attachment %q: %w", trimmed, err)
		}
		if info.IsDir() {
			runCleanups(cleanups, false)
			return nil, nil, fmt.Errorf("imsg: attachment %q is a directory", trimmed)
		}

		if needsSandboxBypass(absolute) {
			copiedPath, cleanup, err := copyToPictures(absolute)
			if err != nil {
				runCleanups(cleanups, false)
				return nil, nil, err
			}
			cleanups = append(cleanups, cleanup)
			absolute = copiedPath
		}

		attachments = append(attachments, attachmentInfo{
			Path:        absolute,
			DelaySecond: attachmentDelaySeconds(info.Size()),
		})
	}

	return attachments, cleanups, nil
}

func expandHome(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("imsg: resolve home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}

	if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("imsg: unsupported tilde path %q", path)
	}

	return path, nil
}

func attachmentDelaySeconds(sizeBytes int64) int {
	sizeMB := float64(sizeBytes) / (1024 * 1024)
	switch {
	case sizeMB < 1:
		return 2
	case sizeMB < 10:
		return 3
	default:
		return 5
	}
}

func needsSandboxBypass(path string) bool {
	sep := string(filepath.Separator)
	allowed := []string{sep + "Pictures" + sep, sep + "Downloads" + sep, sep + "Documents" + sep}
	for _, allow := range allowed {
		if strings.Contains(path, allow) {
			return false
		}
	}
	return true
}

func copyToPictures(path string) (string, func() error, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil, fmt.Errorf("imsg: resolve home directory: %w", err)
	}

	picturesDir := filepath.Join(home, "Pictures")
	if err := os.MkdirAll(picturesDir, 0700); err != nil {
		return "", nil, fmt.Errorf("imsg: ensure Pictures directory: %w", err)
	}

	pattern := tempPatternFor(path)
	tempFile, err := os.CreateTemp(picturesDir, pattern)
	if err != nil {
		return "", nil, fmt.Errorf("imsg: create temp attachment: %w", err)
	}
	tempPath := tempFile.Name()

	cleanup := func() error {
		return os.Remove(tempPath)
	}

	source, err := os.Open(path)
	if err != nil {
		tempFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("imsg: open attachment %q: %w", path, err)
	}
	defer source.Close()

	if _, err := io.Copy(tempFile, source); err != nil {
		tempFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("imsg: copy attachment %q: %w", path, err)
	}

	if err := tempFile.Chmod(0600); err != nil {
		tempFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("imsg: secure temp attachment: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("imsg: finalize temp attachment: %w", err)
	}

	return tempPath, cleanup, nil
}

func tempPatternFor(path string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)

	safeBase := sanitizeSegment(base, 60)
	safeExt := sanitizeSegment(ext, 10)
	if safeExt == "." {
		safeExt = ""
	}

	if safeBase == "" {
		return "imsg_temp_*" + safeExt
	}
	return "imsg_temp_" + safeBase + "_*" + safeExt
}

func sanitizeSegment(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if b.Len() >= maxLen {
			break
		}
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func runCleanups(cleanups []func() error, debug bool) {
	for _, cleanup := range cleanups {
		if cleanup == nil {
			continue
		}
		if err := cleanup(); err != nil && debug {
			fmt.Fprintf(os.Stderr, "imsg: cleanup failed: %v\n", err)
		}
	}
}
