package database

import "github.com/cockroachdb/pebble"

var _ Iterator = (*pebbleIterator)(nil)

type pebbleIterator struct {
	*pebble.Iterator
}

func (pi *pebbleIterator) Release() {
	err := pi.Close()
	if err != nil {
		logger.Criticalf("while closing iterator: %s", err)
	}
}
