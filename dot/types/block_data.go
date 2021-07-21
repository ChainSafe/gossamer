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
	"bytes"
	"fmt"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// BlockData is stored within the BlockDB
type BlockData struct {
	Hash          common.Hash
	Header        *optional.Header
	Body          *optional.Body
	Receipt       *optional.Bytes
	MessageQueue  *optional.Bytes
	Justification *optional.Bytes
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

// Encode performs SCALE encoding of the BlockData
func (bd *BlockData) Encode() ([]byte, error) {
	enc := bd.Hash[:]

	if bd.Header.Exists() {
		//venc, err := scale.Encode(bd.Header.Value())
		head, err := NewHeaderFromOptional(bd.Header)
		if err != nil {
			return nil, err
		}
		venc, err := head.Encode()

		//var venc []byte
		//ph, err := scale2.Marshal(bd.Header.Value().ParentHash)
		//if err != nil {
		//	return nil, err
		//}
		//venc = append(venc, ph...)
		//
		//num, err := scale2.Marshal(bd.Header.Value().Number)
		//if err != nil {
		//	return nil, err
		//}
		//venc = append(venc, num...)
		//
		//sr, err := scale2.Marshal(bd.Header.Value().StateRoot)
		//if err != nil {
		//	return nil, err
		//}
		//venc = append(venc, sr...)
		//
		//er, err := scale2.Marshal(bd.Header.Value().ExtrinsicsRoot)
		//if err != nil {
		//	return nil, err
		//}
		//venc = append(venc, er...)
		//
		//d, err := bd.Header.Value().Digest.Encode()
		//if err != nil {
		//	return nil, err
		//}
		//venc = append(venc, d...)

		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.Body.Exists() {
		venc, err := scale.Marshal(bd.Body.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.Receipt != nil && bd.Receipt.Exists() {
		venc, err := scale.Marshal(bd.Receipt.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.MessageQueue != nil && bd.MessageQueue.Exists() {
		venc, err := scale.Marshal(bd.MessageQueue.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	if bd.Justification != nil && bd.Justification.Exists() {
		venc, err := scale.Marshal(bd.Justification.Value())
		if err != nil {
			return nil, err
		}
		enc = append(enc, byte(1)) // Some
		enc = append(enc, venc...)
	} else {
		enc = append(enc, byte(0)) // None
	}

	return enc, nil
}

// Decode decodes the SCALE encoded input to BlockData
func (bd *BlockData) Decode(r *bytes.Buffer) error {
	hash, err := common.ReadHash(r)
	if err != nil {
		return err
	}
	bd.Hash = hash

	bd.Header, err = decodeOptionalHeader(r)
	if err != nil {
		return err
	}

	bd.Body, err = decodeOptionalBody(r)
	if err != nil {
		return err
	}

	bd.Receipt, err = decodeOptionalBytes(r)
	if err != nil {
		return err
	}

	bd.MessageQueue, err = decodeOptionalBytes(r)
	if err != nil {
		return err
	}

	bd.Justification, err = decodeOptionalBytes(r)
	if err != nil {
		return err
	}

	return nil
}

// EncodeBlockDataArray encodes an array of BlockData using SCALE
func EncodeBlockDataArray(bds []*BlockData) ([]byte, error) {
	enc, err := scale.Marshal(int32(len(bds)))
	if err != nil {
		return nil, err
	}

	for _, bd := range bds {
		benc, err := bd.Encode()
		if err != nil {
			return nil, err
		}
		enc = append(enc, benc...)
	}

	return enc, nil
}

// DecodeBlockDataArray decodes a SCALE encoded BlockData array
func DecodeBlockDataArray(r *bytes.Buffer) ([]*BlockData, error) {
	sd := scale.NewDecoder(r)

	var l int32
	err := sd.Decode(&l)
	if err != nil {
		return nil, err
	}

	length := int(l)
	bds := make([]*BlockData, length)

	for i := 0; i < length; i++ {
		bd := new(BlockData)
		err = bd.Decode(r)
		if err != nil {
			return bds, err
		}

		bds[i] = bd
	}

	return bds, err
}
