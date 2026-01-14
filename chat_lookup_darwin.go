//go:build darwin

package imsg

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const messagesDBRelativePath = "Library/Messages/chat.db"

// LookupChatIDsByName finds chat IDs whose display name matches the provided name.
//
// It returns chat GUIDs suitable for SendChatID.
func LookupChatIDsByName(name string) ([]string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, ErrMissingChatName
	}

	path, err := messagesDBPath()
	if err != nil {
		return nil, err
	}

	if _, statErr := os.Stat(path); statErr != nil {
		return nil, fmt.Errorf("imsg: chat database unavailable: %w", statErr)
	}

	dsn := messagesDBDSN(path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("imsg: open chat database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	query := `
SELECT chat.guid
FROM chat
WHERE chat.display_name LIKE ? ESCAPE '\'
ORDER BY (
    SELECT MAX(message.date)
    FROM chat_message_join cmj
    INNER JOIN message ON message.ROWID = cmj.message_id
    WHERE cmj.chat_id = chat.ROWID
) DESC
`

	pattern := "%" + escapeLike(trimmed) + "%"

	rows, err := db.Query(query, pattern)
	if err != nil {
		return nil, fmt.Errorf("imsg: query chat database: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	chatIDs := make([]string, 0)
	for rows.Next() {
		var guid sql.NullString
		if err := rows.Scan(&guid); err != nil {
			return nil, fmt.Errorf("imsg: scan chat rows: %w", err)
		}
		if guid.Valid && guid.String != "" {
			chatIDs = append(chatIDs, guid.String)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("imsg: read chat rows: %w", err)
	}

	return chatIDs, nil
}

func messagesDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("imsg: resolve home directory: %w", err)
	}

	return filepath.Join(home, messagesDBRelativePath), nil
}

func messagesDBDSN(path string) string {
	uri := url.URL{Scheme: "file", Path: path}
	return fmt.Sprintf("%s?mode=ro&_busy_timeout=5000", uri.String())
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(value)
}
