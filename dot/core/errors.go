package core

import (
	"errors"
	"fmt"
)

var ErrNilBlockState = errors.New("cannot have nil BlockState")
var ErrNilStorageState = errors.New("cannot have nil StorageState")
var ErrNilKeystore = errors.New("cannot have nil keystore")
var ErrNoKeysProvided = errors.New("no keys provided for authority node")
var ErrServiceStopped = errors.New("service has been stopped")
var ErrCannotValidateTx = errors.New("could not validate transaction")

func ErrNilChannel(s string) error {
	return fmt.Errorf("cannot have nil channel %s", s)
}

func ErrMessageCast(s string) error {
	return fmt.Errorf("could not cast network.Message to %s", s)
}

func ErrUnsupportedMsgType(d int) error {
	return fmt.Errorf("received unsupported message type %d", d)
}
