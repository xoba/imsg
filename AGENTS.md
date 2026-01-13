# Agent Notes

This repository is a small macOS iMessage sender for Go. When making changes, keep the API minimal and stable.

## Development

- Format Go code with `gofmt -w`.
- Run checks with `bin/check`.

## Architecture

- AppleScript generation lives in `script.go`.
- Sending and file handling live in `client.go`.
- The CLI tool is `cmd/send-demo`.

## Constraints

- Sending only (no message receiving).
- macOS only.
- Keep defaults and behavior predictable.
