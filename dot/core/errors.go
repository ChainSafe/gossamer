package core

import (
	"errors"
	"fmt"
)

var ErrNilBlockState = errors.New("cannot have nil BlockState")
var ErrServiceStopped = errors.New("service has been stopped")

func ErrNilChannel(s string) error {
	return fmt.Errorf("cannot have nil channel %s", s)
}
