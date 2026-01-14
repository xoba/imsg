package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"xoba.com/imsg"
)

func main() {
	name := flag.String("name", "", "chat display name to search for")
	limit := flag.Int("limit", 0, "max results to print (0 for all)")
	flag.Parse()

	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(os.Stderr, "imsg-lookup: -name is required")
		flag.Usage()
		os.Exit(2)
	}
	if *limit < 0 {
		fmt.Fprintln(os.Stderr, "imsg-lookup: -limit must be >= 0")
		flag.Usage()
		os.Exit(2)
	}

	chatIDs, err := imsg.LookupChatIDsByName(*name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *limit > 0 && len(chatIDs) > *limit {
		chatIDs = chatIDs[:*limit]
	}

	for _, chatID := range chatIDs {
		fmt.Println(chatID)
	}
}
