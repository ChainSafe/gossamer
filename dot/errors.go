package dot

import (
	"errors"
)

// ErrNoKeysProvided is returned when no keys are given for an authority node
var ErrNoKeysProvided = errors.New("no keys provided for authority node")

// ErrInvalidKeystoreType when trying to create a service with the wrong keystore type
var ErrInvalidKeystoreType = errors.New("invalid keystore type")
