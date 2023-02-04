// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package badger

import (
	"errors"
	"fmt"
	"path/filepath"
)

// Settings is the database settings.
type Settings struct {
	// Path is the database directory path to use.
	// Note it should be the empty string if InMemory is true.
	// It defaults to the empty string if left unset.
	Path *string
	// InMemory is whether to use an in-memory database.
	InMemory *bool
}

// WithPath sets the path in the settings.
func (s *Settings) WithPath(path string) {
	s.Path = ptrTo(path)
}

// WithInMemory sets the in memory flag in the settings.
func (s *Settings) WithInMemory(inMemory bool) {
	s.InMemory = ptrTo(inMemory)
}

// SetDefaults sets the default values on the settings.
func (s *Settings) SetDefaults() {
	if s.Path == nil {
		s.Path = ptrTo("")
	}

	if s.InMemory == nil {
		s.InMemory = ptrTo(false)
	}
}

var (
	ErrPathSetInMemory = errors.New("path set with database in-memory")
)

// Validate validates the settings.
func (s *Settings) Validate() (err error) { //skipcq: GO-W1029
	if *s.InMemory {
		if *s.Path != "" {
			// Note badger v3 enforces the path is not set in this case.
			return fmt.Errorf("%w: %q", ErrPathSetInMemory, *s.Path)
		}
	} else {
		_, err = filepath.Abs(*s.Path)
		if err != nil {
			return fmt.Errorf("changing path to absolute path: %w", err)
		}
	}

	return nil
}
