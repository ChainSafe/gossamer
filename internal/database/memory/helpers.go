// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package memory

func copyBytes(b []byte) (bCopy []byte) {
	bCopy = make([]byte, len(b))
	copy(bCopy, b)
	return bCopy
}

func (db *Database) panicOnClosed() {
	if db.closed {
		panic("database is closed")
	}
}
