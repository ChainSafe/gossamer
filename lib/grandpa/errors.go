package grandpa

import (
	"errors"
)

var ErrBlockDoesNotExist = errors.New("block does not exist")
var ErrInvalidSignature = errors.New("signature is not valid")
var ErrSetIDMismatch = errors.New("set IDs do not match")
var ErrEquivocation = errors.New("vote is equivocatory")
