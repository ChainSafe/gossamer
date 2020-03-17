package state

import (
	"bytes"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
)

// SetReceipt sets a Receipt in the database
func (bs *BlockState) SetReceipt(hash common.Hash, data []byte) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	err := bs.db.Put(prefixHash(hash, receiptPrefix), data)
	if err != nil {
		return err
	}

	return nil
}

// GetReceipt retrieves a Receipt from the database
func (bs *BlockState) GetReceipt(hash common.Hash) (*optional.Bytes, error) {
	data, err := bs.db.Get(prefixHash(hash, receiptPrefix))
	if err != nil {
		return nil, err
	}

	r := &bytes.Buffer{}
	_, err = r.Write(data)
	if err != nil {
		return nil, err
	}

	return optional.NewBytes(true, data), nil
}

// SetMessageQueue sets a MessageQueue in the database
func (bs *BlockState) SetMessageQueue(hash common.Hash, data []byte) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	err := bs.db.Put(prefixHash(hash, messageQueuePrefix), data)
	if err != nil {
		return err
	}

	return nil
}

// GetMessageQueue retrieves a MessageQueue from the database
func (bs *BlockState) GetMessageQueue(hash common.Hash) (*optional.Bytes, error) {
	data, err := bs.db.Get(prefixHash(hash, messageQueuePrefix))
	if err != nil {
		return nil, err
	}

	r := &bytes.Buffer{}
	_, err = r.Write(data)
	if err != nil {
		return nil, err
	}

	return optional.NewBytes(true, data), nil
}

// SetJustification sets a Justification in the database
func (bs *BlockState) SetJustification(hash common.Hash, data []byte) error {
	bs.lock.Lock()
	defer bs.lock.Unlock()

	err := bs.db.Put(prefixHash(hash, justificationPrefix), data)
	if err != nil {
		return err
	}

	return nil
}

// GetJustification retrieves a Justification from the database
func (bs *BlockState) GetJustification(hash common.Hash) (*optional.Bytes, error) {
	data, err := bs.db.Get(prefixHash(hash, justificationPrefix))
	if err != nil {
		return nil, err
	}

	r := &bytes.Buffer{}
	_, err = r.Write(data)
	if err != nil {
		return nil, err
	}

	return optional.NewBytes(true, data), nil
}

// prefixHash = prefix + hash
func prefixHash(hash common.Hash, prefix []byte) []byte {
	return append(prefix, hash.ToBytes()...)
}
