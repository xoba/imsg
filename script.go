package imsg

import (
	"fmt"
	"strings"
)

type attachmentInfo struct {
	Path        string
	DelaySecond int
}

func buildSendScript(recipient, text string, attachments []attachmentInfo) string {
	escapedRecipient := escapeAppleScriptString(recipient)

	var b strings.Builder
	b.WriteString("tell application \"Messages\"\n")
	b.WriteString("    set targetService to 1st service whose service type is iMessage and enabled is true\n")
	b.WriteString(fmt.Sprintf("    set targetBuddy to buddy \"%s\" of targetService\n", escapedRecipient))

	if text != "" {
		escapedText := escapeAppleScriptString(text)
		b.WriteString(fmt.Sprintf("    send \"%s\" to targetBuddy\n", escapedText))
	}

	for _, attachment := range attachments {
		escapedPath := escapeAppleScriptString(attachment.Path)
		b.WriteString(fmt.Sprintf("    send POSIX file \"%s\" to targetBuddy\n", escapedPath))
		if attachment.DelaySecond > 0 {
			b.WriteString(fmt.Sprintf("    delay %d\n", attachment.DelaySecond))
		}
	}

	b.WriteString("end tell")
	return b.String()
}

func escapeAppleScriptString(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	return replacer.Replace(value)
}
