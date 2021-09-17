package main

import (
	"os"

	"github.com/ChainSafe/gossamer/lib/utils"
)

func main() {
	title := os.Getenv("RAW_TITLE")
	body := os.Getenv("RAW_BODY")
	err := utils.CheckPRDescription(title, body)
	if err != nil {
		os.Exit(1)
	}
}
