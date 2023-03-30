// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	"fmt"
	"path/filepath"
)

// Settings is the database settings.
type Settings struct {
	// Path is the database directory path to use.
	// It defaults to the current directory if left unset.
	Path string
	// InMemory is whether to use an in-memory database.
	InMemory *bool
}

// SetDefaults sets the default values on the settings.
func (s *Settings) SetDefaults() {
	if s.Path == "" {
		s.Path = "."
	}

	if s.InMemory == nil {
		s.InMemory = new(bool)
	}
}

// Validate validates the settings.
func (s Settings) Validate() (err error) { //skipcq: GO-W1029
	_, err = filepath.Abs(s.Path)
	if err != nil {
		return fmt.Errorf("changing path to absolute path: %w", err)
	}

	return nil
}
