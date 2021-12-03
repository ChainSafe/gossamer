// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package types

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
)

// BlockData is stored within the BlockDB
// The BlockData fields are optionals and thus are represented as pointers to ensure
// correct encoding
type BlockData struct {
	Hash          common.Hash
	Header        *Header
	Body          *Body
	Receipt       *[]byte
	MessageQueue  *[]byte
	Justification *[]byte
}

// NewEmptyBlockData Creates an empty blockData struct
func NewEmptyBlockData() *BlockData {
	return &BlockData{}
}

// Number returns the BlockNumber of the BlockData's header, nil if it doesn't exist
func (bd *BlockData) Number() *big.Int {
	if bd == nil || bd.Header == nil {
		return nil
	}

	return bd.Header.Number
}

func (bd *BlockData) String() string {
	str := fmt.Sprintf("Hash=%s ", bd.Hash)

	if bd.Header != nil {
		str = str + fmt.Sprintf("Header=%s ", bd.Header)
	}

	if bd.Body != nil {
		str = str + fmt.Sprintf("Body=%s ", *bd.Body)
	}

	if bd.Receipt != nil {
		str = str + fmt.Sprintf("Receipt=0x%x ", bd.Receipt)
	}

	if bd.MessageQueue != nil {
		str = str + fmt.Sprintf("MessageQueue=0x%x ", bd.MessageQueue)
	}

	if bd.Justification != nil {
		str = str + fmt.Sprintf("Justification=0x%x ", bd.Justification)
	}

	return str
}
