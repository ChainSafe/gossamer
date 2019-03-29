package polkadb

import "github.com/pkg/errors"

var (
	// ErrClosedDB indicates the database did not open
	ErrClosedDB = errors.New("database did not open ")

	// ErrRefusedClose indicates the database refused to close
	ErrRefusedClose = errors.New("failed to close database ")

	// ErrKeyRetrieval indicates a failure to retrieve a key
	ErrKeyRetrieval = errors.New("failed  to retrieve key ")

	// ErrValueRetrieval indicates a failure to retrieve a value
	ErrValueRetrieval = errors.New("failed  to retrieve value ")

	// ErrDecoding indicates a failure to decode key
	ErrDecoding = errors.New("failed to decode ")

	// ErrBatchWrites indicates a failure to write batch txs
	ErrBatchWrites = errors.New("failed to batch write txs ")

	// ErrBatchFlush indicates an error with stored writeBatch
	ErrBatchFlush = errors.New("failed to flush ")

	// ErrBatchDeletion indicates a failure to delete a key
	ErrBatchDeletion = errors.New("failed to delete batch key ")
)
