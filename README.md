# imsg

A small iMessage sender for Go on macOS. It sends text and attachments via AppleScript and keeps the API minimal and stable.

- Send text and files to phone numbers or emails
- No message receiving
- No external dependencies

## Install

```bash
go get xoba.com/imsg
```

## Usage

Send text:

```go
package main

import "xoba.com/imsg"

func main() {
    _ = imsg.SendText("+12025550123", "Hello from Go")
}
```

Send a file:

```go
package main

import "xoba.com/imsg"

func main() {
    _ = imsg.SendFile("+12025550123", "/path/to/video.mp4")
}
```

Send both text and attachments:

```go
package main

import "xoba.com/imsg"

func main() {
    _ = imsg.Send("+12025550123", imsg.Message{
        Text: "check this out",
        Attachments: []string{
            "/path/to/image.png",
            "/path/to/video.mp4",
        },
    })
}
```

## Group chats

Look up chat IDs by name, then send to the chat ID:

```go
package main

import (
    "log"

    "xoba.com/imsg"
)

func main() {
    chatIDs, err := imsg.LookupChatIDsByName("stonks")
    if err != nil {
        log.Fatal(err)
    }
    if len(chatIDs) == 0 {
        log.Fatal("no matching chats")
    }

    _ = imsg.SendChatID(chatIDs[0], imsg.Message{
        Text: "hello group",
    })
}
```

## CLI

A simple sender lives in `cmd/send-demo`:

```bash
go run ./cmd/send-demo -to "+12025550123" -text "hello" -file "/path/to/file.png"
```

## Requirements

- macOS with Messages.app signed in to iMessage
- Allow your terminal/IDE to control Messages when macOS prompts (Privacy & Security -> Automation)
- Grant Full Disk Access if you use `LookupChatIDsByName` (Privacy & Security -> Full Disk Access)

Attachments outside `~/Pictures`, `~/Downloads`, or `~/Documents` are copied to a temporary file in `~/Pictures` so Messages can access them. The temporary file is deleted after the send completes.

## Public API

- `Version`
- `Message`
- `Client`, `NewClient`, `WithTimeout`, `WithDebug`
- `Send`, `SendText`, `SendFile`, `SendFiles`, `SendChatID`
- `LookupChatIDsByName`
- Exported errors: `ErrUnsupportedOS`, `ErrMissingRecipient`, `ErrMissingChatID`, `ErrMissingChatName`, `ErrEmptyMessage`, `ErrScriptTimeout`, `ErrAttachmentPathEmpty`, `ErrAttachmentIsDirectory`, `ErrUnsupportedTildePath`

## How it works

The library generates AppleScript that targets Messages.app and executes it with `osascript`. This is the standard automation approach on macOS for sending iMessage content.

## License

MIT
