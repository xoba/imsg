package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"xoba.com/imsg"
)

func main() {
	to := flag.String("to", "", "recipient phone number or email")
	text := flag.String("text", "", "message text")
	file := flag.String("file", "", "path to a file to send")
	debug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	if *to == "" {
		fmt.Fprintln(os.Stderr, "imsg-send: -to is required")
		flag.Usage()
		os.Exit(2)
	}
	if strings.TrimSpace(*text) == "" && *file == "" {
		fmt.Fprintln(os.Stderr, "imsg-send: provide -text and/or -file")
		flag.Usage()
		os.Exit(2)
	}

	client := imsg.NewClient(imsg.WithDebug(*debug))

	msg := imsg.Message{Text: *text}
	if *file != "" {
		msg.Attachments = []string{*file}
	}

	if err := client.Send(*to, msg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
