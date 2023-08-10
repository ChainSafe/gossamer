// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"bytes"
)

var _ Batch = (*tableBatch)(nil)

type tableBatch struct {
	batch  Batch
	prefix []byte
}

func (tb *tableBatch) Put(key, value []byte) error {
	tableItemKey := bytes.Join([][]byte{tb.prefix, key}, nil)
	return tb.batch.Put(tableItemKey, value)
}

func (tb *tableBatch) Del(key []byte) error {
	tableItemKey := bytes.Join([][]byte{tb.prefix, key}, nil)
	return tb.batch.Del(tableItemKey)
}

func (tb *tableBatch) Flush() error {
	return tb.batch.Flush()
}

func (tb *tableBatch) ValueSize() int {
	return tb.batch.ValueSize()
}

func (tb *tableBatch) Reset() {
	tb.batch.Reset()
}

func (tb *tableBatch) Close() error {
	return tb.batch.Close()
}
