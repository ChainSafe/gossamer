// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pprof

import "fmt"

// Settings are the settings for the Pprof service.
type Settings struct {
	// ListeningAddress is the HTTP pprof server
	// listening address.
	ListeningAddress string
	// See runtime.SetBlockProfileRate
	// Set to 0 to disable profiling.
	BlockProfileRate int
	// See runtime.SetMutexProfileFraction
	// Set to 0 to disable profiling.
	MutexProfileRate int
}

func (s *Settings) setDefaults() { //skipcq: GO-W1029
	if s.ListeningAddress == "" {
		s.ListeningAddress = "localhost:6060"
	}
}

func (s Settings) String() string { //skipcq: GO-W1029
	return fmt.Sprintf(
		"listening on %s and setting block profile rate to %d, mutex profile rate to %d",
		s.ListeningAddress, s.BlockProfileRate, s.MutexProfileRate)
}
