package node

import "errors"

type writeCall struct {
	written []byte
	n       int
	err     error
}

var errTest = errors.New("test error")
