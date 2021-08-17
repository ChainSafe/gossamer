// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
)

// BlockData is stored within the BlockDB
type BlockDataVdt struct {
	Hash          common.Hash
	Header        *HeaderVdt
	Body          *Body
	Receipt       *[]byte
	MessageQueue  *[]byte
	Justification *[]byte
}

// BlockData is stored within the BlockDB
type BlockData struct {
	Hash          common.Hash
	Header        *optional.Header
	Body          *optional.Body
	Receipt       *optional.Bytes
	MessageQueue  *optional.Bytes
	Justification *optional.Bytes
}

func NewEmptyBlockDataVdt() *BlockDataVdt{
	bd := &BlockDataVdt{
		Header:        NewEmptyHeaderVdt(),
		Body:          nil,
		Receipt:       nil,
		MessageQueue:  nil,
		Justification: nil,
	}
	return bd
}

// Number returns the BlockNumber of the BlockData's header, nil if it doesn't exist
func (bd *BlockData) Number() *big.Int {
	if bd == nil || bd.Header == nil || !bd.Header.Exists() {
		return nil
	}

	return bd.Header.Value().Number
}

func (bd *BlockData) String() string {
	str := fmt.Sprintf("Hash=%s ", bd.Hash)

	if bd.Header != nil && bd.Header.Exists() {
		str = str + fmt.Sprintf("Header=%s ", bd.Header)
	}

	if bd.Body != nil && bd.Body.Exists() {
		str = str + fmt.Sprintf("Body=%s ", bd.Body)
	}

	if bd.Receipt != nil && bd.Receipt.Exists() {
		str = str + fmt.Sprintf("Receipt=0x%x ", bd.Receipt)
	}

	if bd.MessageQueue != nil && bd.MessageQueue.Exists() {
		str = str + fmt.Sprintf("MessageQueue=0x%x ", bd.MessageQueue)
	}

	if bd.Justification != nil && bd.Justification.Exists() {
		str = str + fmt.Sprintf("Justification=0x%x ", bd.Justification)
	}

	return str
}

func (bd *BlockDataVdt) Number() *big.Int {
	if bd == nil || bd.Header == nil {
		return nil
	}

	return bd.Header.Number
}

func (bd *BlockDataVdt) String() string {
	str := fmt.Sprintf("Hash=%s ", bd.Hash)

	if bd.Header != nil {
		str = str + fmt.Sprintf("Header=%s ", bd.Header)
	}

	if bd.Body != nil {
		str = str + fmt.Sprintf("Body=%s ", bd.Body)
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