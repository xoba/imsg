package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"xoba.com/imsg"
)

func main() {
	chatID := flag.String("chat-id", "", "chat id (guid) from Messages database")
	text := flag.String("text", "", "message text")
	file := flag.String("file", "", "path to a file to send")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	if strings.TrimSpace(*chatID) == "" {
		fmt.Fprintln(os.Stderr, "imsg-send-chat: -chat-id is required")
		flag.Usage()
		os.Exit(2)
	}
	if strings.TrimSpace(*text) == "" && *file == "" {
		fmt.Fprintln(os.Stderr, "imsg-send-chat: provide -text and/or -file")
		flag.Usage()
		os.Exit(2)
	}

	client := imsg.NewClient(imsg.WithDebug(*debug))

	msg := imsg.Message{Text: *text}
	if *file != "" {
		msg.Attachments = []string{*file}
	}

	if err := client.SendChatID(*chatID, msg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
