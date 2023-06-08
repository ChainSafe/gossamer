// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package erasure

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/klauspost/reedsolomon"
)

// ErrNotEnoughValidators cannot encode something for zero or one validator
var ErrNotEnoughValidators = errors.New("expected at least 2 validators")

// ObtainChunks obtains erasure-coded chunks, divides data into number of validatorsQty chunks and
// creates parity chunks for reconstruction
func ObtainChunks(validatorsQty int, data []byte) ([][]byte, error) {
	recoveryThres, err := recoveryThreshold(validatorsQty)
	if err != nil {
		return nil, err
	}
	enc, err := reedsolomon.New(validatorsQty, recoveryThres)
	if err != nil {
		return nil, fmt.Errorf("creating new reed solomon failed: %w", err)
	}
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}
	err = enc.Encode(shards)
	if err != nil {
		return nil, err
	}

	return shards, nil
}

// Reconstruct the missing data from a set of chunks
func Reconstruct(validatorsQty, originalDataLen int, chunks [][]byte) ([]byte, error) {
	recoveryThres, err := recoveryThreshold(validatorsQty)
	if err != nil {
		return nil, err
	}

	enc, err := reedsolomon.New(validatorsQty, recoveryThres)
	if err != nil {
		return nil, err
	}
	err = enc.Reconstruct(chunks)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err = enc.Join(buf, chunks, originalDataLen)
	return buf.Bytes(), err
}

// recoveryThreshold gives the max number of shards/chunks that we can afford to lose and still construct
// the full initial data.  Total number of chunks will be validatorQty + recoveryThreshold
func recoveryThreshold(validators int) (int, error) {
	if validators <= 1 {
		return 0, ErrNotEnoughValidators
	}

	needed := (validators - 1) / 3

	return needed + 1, nil
}
