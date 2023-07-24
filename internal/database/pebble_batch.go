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
	err := pb.batch.Set(key, value, &pebble.WriteOptions{})
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
