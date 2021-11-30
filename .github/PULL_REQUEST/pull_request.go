// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"fmt"
	"os"

	"github.com/ChainSafe/gossamer/lib/utils"
)

func main() {
	title := os.Getenv("RAW_TITLE")
	body := os.Getenv("RAW_BODY")
	err := utils.CheckPRDescription(title, body)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
