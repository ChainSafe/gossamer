package grandpa

import (
	"errors"
)

var ErrBlockDoesNotExist = errors.New("block does not exist")
var ErrInvalidSignature = errors.new("signature is not valid")
var ErrSetIDMismatch = errors.New("set IDs do not match")
