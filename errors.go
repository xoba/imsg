package imsg

import "errors"

var (
	// ErrUnsupportedOS is returned when running on a non-macOS platform.
	ErrUnsupportedOS = errors.New("imsg: only supported on darwin")
	// ErrMissingRecipient is returned when the recipient is empty.
	ErrMissingRecipient = errors.New("imsg: recipient is required")
	// ErrEmptyMessage is returned when both text and attachments are empty.
	ErrEmptyMessage = errors.New("imsg: message text or attachments required")
)
