// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package state

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"golang.org/x/exp/slices"
)

// prefixKey = prefix + hash
func prefixKey(hash common.Hash, prefix []byte) []byte {
	return append(prefix, hash.ToBytes()...)
}

// HasReceipt returns if the db contains a receipt at the given hash
func (bs *BlockState) HasReceipt(hash common.Hash) (bool, error) {
	return bs.db.Has(prefixKey(hash, receiptPrefix))
}

// SetReceipt sets a Receipt in the database
func (bs *BlockState) SetReceipt(hash common.Hash, data []byte) error {
	err := bs.db.Put(prefixKey(hash, receiptPrefix), data)
	if err != nil {
		return err
	}

	return nil
}

// GetReceipt retrieves a Receipt from the database
func (bs *BlockState) GetReceipt(hash common.Hash) ([]byte, error) {
	data, err := bs.db.Get(prefixKey(hash, receiptPrefix))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// HasMessageQueue returns if the db contains a MessageQueue at the given hash
func (bs *BlockState) HasMessageQueue(hash common.Hash) (bool, error) {
	return bs.db.Has(prefixKey(hash, messageQueuePrefix))
}

// SetMessageQueue sets a MessageQueue in the database
func (bs *BlockState) SetMessageQueue(hash common.Hash, data []byte) error {
	err := bs.db.Put(prefixKey(hash, messageQueuePrefix), data)
	if err != nil {
		return err
	}

	return nil
}

// GetMessageQueue retrieves a MessageQueue from the database
func (bs *BlockState) GetMessageQueue(hash common.Hash) ([]byte, error) {
	data, err := bs.db.Get(prefixKey(hash, messageQueuePrefix))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// HasJustification returns if the db contains a Justification at the given hash
func (bs *BlockState) HasJustification(hash common.Hash) (bool, error) {
	return bs.db.Has(prefixKey(hash, justificationPrefix))
}

// SetJustification sets a Justification in the database
func (bs *BlockState) SetJustification(hash common.Hash, data []byte) error {
	err := bs.db.Put(prefixKey(hash, justificationPrefix), data)
	if err != nil {
		return err
	}

	return nil
}

// GetJustification retrieves a Justification from the database
func (bs *BlockState) GetJustification(hash common.Hash) ([]byte, error) {
	data, err := bs.db.Get(prefixKey(hash, justificationPrefix))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// GetAuthoritesChangesFromBlock retrieves blocks numbers where authority set changes happened
func (bs *BlockState) GetAuthoritesChangesFromBlock(initialBlockNumber uint) ([]uint, error) {
	blockNumbers := make([]uint, 0)
	iter, err := bs.db.NewPrefixIterator(setIDChangePrefix)
	if err != nil {
		return nil, err
	}

	for iter.Next() {
		blockNumber := common.BytesToUint(iter.Value())
		if blockNumber >= initialBlockNumber {
			blockNumbers = append(blockNumbers, blockNumber)
		}
	}

	// To ensure the order of the blocks
	slices.Sort(blockNumbers)
	return blockNumbers, nil
}
