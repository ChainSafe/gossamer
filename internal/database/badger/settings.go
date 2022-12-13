package badger

import (
	"fmt"
	"path/filepath"
)

// Settings is the database settings.
type Settings struct {
	// Path is the database directory path to use.
	// It defaults to the current directory if left unset.
	Path *string
}

// SetDefaults sets the default values on the settings.
func (s *Settings) SetDefaults() {
	if s.Path == nil {
		s.Path = new(string)
	}
}

// Validate validates the settings.
func (s Settings) Validate() (err error) {
	_, err = filepath.Abs(*s.Path)
	if err != nil {
		return fmt.Errorf("changing path to absolute path: %w", err)
	}

	return nil
}
