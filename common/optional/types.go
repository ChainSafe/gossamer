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

package optional

import (
	"fmt"
	"math/big"

	common "github.com/ChainSafe/gossamer/common"
)

type Uint32 struct {
	exists bool
	value  uint32
}

func NewUint32(exists bool, value uint32) *Uint32 {
	return &Uint32{
		exists: exists,
		value:  value,
	}
}

func (x *Uint32) Exists() bool {
	return x.exists
}

func (x *Uint32) Value() uint32 {
	return x.value
}

func (x *Uint32) String() string {
	return fmt.Sprintf("%d", x.value)
}

func (x *Uint32) Set(exists bool, value uint32) {
	x.exists = exists
	x.value = value
}

type Bytes struct {
	exists bool
	value  []byte
}

func NewBytes(exists bool, value []byte) *Bytes {
	return &Bytes{
		exists: exists,
		value:  value,
	}
}

func (x *Bytes) Exists() bool {
	return x.exists
}

func (x *Bytes) Value() []byte {
	return x.value
}

func (x *Bytes) String() string {
	return fmt.Sprintf("%x", x.value)
}

func (x *Bytes) Set(exists bool, value []byte) {
	x.exists = exists
	x.value = value
}

type Hash struct {
	exists bool
	value  common.Hash
}

func NewHash(exists bool, value common.Hash) *Hash {
	return &Hash{
		exists: exists,
		value:  value,
	}
}

func (x *Hash) Exists() bool {
	return x.exists
}

func (x *Hash) Value() common.Hash {
	return x.value
}

func (x *Hash) String() string {
	return fmt.Sprintf("%x", x.value)
}

func (x *Hash) Set(exists bool, value common.Hash) {
	x.exists = exists
	x.value = value
}

// CoreHeader is a state block header
// This is copied from core/types since core/types imports this package, we cannot import core/types.
type CoreHeader struct {
	ParentHash     common.Hash `json:"parentHash"`
	Number         *big.Int    `json:"number"`
	StateRoot      common.Hash `json:"stateRoot"`
	ExtrinsicsRoot common.Hash `json:"extrinsicsRoot"`
	Digest         [][]byte    `json:"digest"`
	//hash           common.Hash
}

type Header struct {
	exists bool
	value  *CoreHeader
}

func NewHeader(exists bool, value *CoreHeader) *Header {
	return &Header{
		exists: exists,
		value:  value,
	}
}

func (x *Header) Exists() bool {
	return x.exists
}

func (x *Header) Value() *CoreHeader {
	return x.value
}

func (x *Header) String() string {
	return fmt.Sprintf("%v", x.value)
}

func (x *Header) Set(exists bool, value *CoreHeader) {
	x.exists = exists
	x.value = value
}

// CoreBody is the extrinsics inside a state block
type CoreBody []byte

// Body represents an optional types.Body.
// The fields need to be exported since it's JSON encoded by the state service.
// TODO: when we change the state service's encoding to SCALE, these fields should become unexported.
type Body struct {
	Exists bool
	Value  *CoreBody
}

func NewBody(exists bool, value *CoreBody) *Body {
	return &Body{
		Exists: exists,
		Value:  value,
	}
}

// func (x *Body) Exists() bool {
// 	return x.Exists
// }

// func (x *Body) Value() *CoreBody {
// 	return x.Value
// }

func (x *Body) String() string {
	return fmt.Sprintf("%v", x.Value)
}

func (x *Body) Set(exists bool, value *CoreBody) {
	x.Exists = exists
	x.Value = value
}
