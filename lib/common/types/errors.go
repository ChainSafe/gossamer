package types

import "errors"

// ErrInvalidResult is returned when decoding a Result type fails
var ErrInvalidResult = errors.New("decoding failed, invalid Result")
