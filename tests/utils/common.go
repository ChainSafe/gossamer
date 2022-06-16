// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package utils

import (
	"os"
)

var (
	// MODE is the value for the environnent variable MODE.
	MODE = os.Getenv("MODE")

	// PORT is the value for the environnent variable PORT.
	PORT = os.Getenv("PORT")

	// LOGLEVEL is the value for the environnent variable LOGLEVEL.
	LOGLEVEL = os.Getenv("LOG")
)
