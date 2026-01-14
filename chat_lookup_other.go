//go:build !darwin

package imsg

// LookupChatIDsByName is unavailable on non-macOS platforms.
func LookupChatIDsByName(name string) ([]string, error) {
	_ = name
	return nil, ErrUnsupportedOS
}
