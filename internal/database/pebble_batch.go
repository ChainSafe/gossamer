// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package database

import (
	"fmt"

	"github.com/cockroachdb/pebble"
)

var _ Batch = (*pebbleBatch)(nil)

type pebbleBatch struct {
	batch *pebble.Batch
}

func (pb *pebbleBatch) Put(key, value []byte) error {
	err := pb.batch.Set(key, value, nil)
	if err != nil {
		return fmt.Errorf("setting to batch writer: %w", err)
	}
	return nil
}
func (pb *pebbleBatch) Del(key []byte) error {
	err := pb.batch.Delete(key, &pebble.WriteOptions{})
	if err != nil {
		return fmt.Errorf("setting to batch delete: %w", err)
	}
	return nil
}

func (pb *pebbleBatch) Flush() error {
	err := pb.batch.Commit(&pebble.WriteOptions{})
	if err != nil {
		return fmt.Errorf("committing batch: %w", err)
	}

	return nil
}

func (pb *pebbleBatch) ValueSize() int {
	return int(pb.batch.Count())
}

func (pb *pebbleBatch) Reset() {
	pb.batch.Reset()
}

func (pb *pebbleBatch) Close() error {
	return pb.batch.Close()
}
