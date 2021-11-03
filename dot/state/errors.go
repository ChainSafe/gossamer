package state

import "errors"

var (
	// ErrBlockHeaderNumberIsNil is returned if the block header is nil
	ErrBlockHeaderNumberIsNil = errors.New("block header number field is nil")
)
