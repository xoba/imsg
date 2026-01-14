package imsg

import "errors"

var (
	// ErrUnsupportedOS is returned when running on a non-macOS platform.
	ErrUnsupportedOS = errors.New("imsg: only supported on darwin")
	// ErrMissingRecipient is returned when the recipient is empty.
	ErrMissingRecipient = errors.New("imsg: recipient is required")
	// ErrEmptyMessage is returned when both text and attachments are empty.
	ErrEmptyMessage = errors.New("imsg: message text or attachments required")
	// ErrMissingChatID is returned when the chat id is empty.
	ErrMissingChatID = errors.New("imsg: chat id is required")
	// ErrMissingChatName is returned when the chat name is empty.
	ErrMissingChatName = errors.New("imsg: chat name is required")
	// ErrScriptTimeout is returned when AppleScript execution exceeds the timeout.
	ErrScriptTimeout = errors.New("imsg: osascript timeout")
	// ErrAttachmentPathEmpty is returned when an attachment path is empty.
	ErrAttachmentPathEmpty = errors.New("imsg: attachment path is empty")
	// ErrAttachmentIsDirectory is returned when an attachment path is a directory.
	ErrAttachmentIsDirectory = errors.New("imsg: attachment is a directory")
	// ErrUnsupportedTildePath is returned when a tilde path cannot be expanded.
	ErrUnsupportedTildePath = errors.New("imsg: unsupported tilde path")
)
