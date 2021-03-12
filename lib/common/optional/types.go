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
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
)

const none = "None"

// Uint32 represents an optional uint32 type.
type Uint32 struct {
	exists bool
	value  uint32
}

// NewUint32 returns a new optional.Uint32
func NewUint32(exists bool, value uint32) *Uint32 {
	return &Uint32{
		exists: exists,
		value:  value,
	}
}

// Exists returns true if the value is Some, false if it is None.
func (x *Uint32) Exists() bool {
	if x == nil {
		return false
	}

	return x.exists
}

// Value returns the uint32 value. It returns 0 if it is None.
func (x *Uint32) Value() uint32 {
	if x == nil {
		return 0
	}

	return x.value
}

// String returns the value as a string.
func (x *Uint32) String() string {
	if x == nil {
		return ""
	}
	if !x.exists {
		return none
	}
	return fmt.Sprintf("%d", x.value)
}

// Set sets the exists and value fields.
func (x *Uint32) Set(exists bool, value uint32) {
	x.exists = exists
	x.value = value
}

// Encode returns the SCALE encoding of the optional.Uint32
func (x *Uint32) Encode() []byte {
	if !x.exists {
		return []byte{0}
	}

	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, x.value)
	return append([]byte{1}, buf...)
}

// Bytes represents an optional Bytes type.
type Bytes struct {
	exists bool
	value  []byte
}

// NewBytes returns a new optional.Bytes
func NewBytes(exists bool, value []byte) *Bytes {
	return &Bytes{
		exists: exists,
		value:  value,
	}
}

// Exists returns true if the value is Some, false if it is None.
func (x *Bytes) Exists() bool {
	return x.exists
}

// Value returns the []byte value. It returns nil if it is None.
func (x *Bytes) Value() []byte {
	return x.value
}

// String returns the value as a string.
func (x *Bytes) String() string {
	if !x.exists {
		return none
	}
	return fmt.Sprintf("%x", x.value)
}

// Set sets the exists and value fields.
func (x *Bytes) Set(exists bool, value []byte) {
	x.exists = exists
	x.value = value
}

// Encode returns the SCALE encoded optional
func (x *Bytes) Encode() ([]byte, error) {
	if x == nil || !x.exists {
		return []byte{0}, nil
	}

	value, err := scale.Encode(x.value)
	if err != nil {
		return nil, err
	}

	return append([]byte{1}, value...), nil
}

// Decode return an optional Bytes from scale encoded data
func (x *Bytes) Decode(r io.Reader) (*Bytes, error) {
	exists, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	if exists > 1 {
		return nil, ErrInvalidOptional
	}

	x.exists = (exists != 0)

	if x.exists {
		sd := scale.Decoder{Reader: r}
		value, err := sd.DecodeByteArray()
		if err != nil {
			return nil, err
		}
		x.value = value
	}

	return x, nil
}

// FixedSizeBytes represents an optional FixedSizeBytes type. It does not length-encode the value when encoding.
type FixedSizeBytes struct {
	exists bool
	value  []byte
}

// NewFixedSizeBytes returns a new optional.FixedSizeBytes
func NewFixedSizeBytes(exists bool, value []byte) *FixedSizeBytes {
	return &FixedSizeBytes{
		exists: exists,
		value:  value,
	}
}

// Exists returns true if the value is Some, false if it is None.
func (x *FixedSizeBytes) Exists() bool {
	return x.exists
}

// Value returns the []byte value. It returns nil if it is None.
func (x *FixedSizeBytes) Value() []byte {
	return x.value
}

// String returns the value as a string.
func (x *FixedSizeBytes) String() string {
	if !x.exists {
		return none
	}
	return fmt.Sprintf("%x", x.value)
}

// Set sets the exists and value fields.
func (x *FixedSizeBytes) Set(exists bool, value []byte) {
	x.exists = exists
	x.value = value
}

// Encode returns the SCALE encoded optional
func (x *FixedSizeBytes) Encode() ([]byte, error) {
	if x == nil || !x.exists {
		return []byte{0}, nil
	}

	return append([]byte{1}, x.value...), nil
}

// Decode return an optional FixedSizeBytes from scale encoded data
func (x *FixedSizeBytes) Decode(r io.Reader) (*FixedSizeBytes, error) {
	exists, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	if exists > 1 {
		return nil, ErrInvalidOptional
	}

	x.exists = (exists != 0)

	if x.exists {
		value, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		x.value = value
	}

	return x, nil
}

// Boolean represents an optional bool type.
type Boolean struct {
	exists bool
	value  bool
}

// NewBoolean returns a new optional.Boolean
func NewBoolean(exists bool, value bool) *Boolean {
	return &Boolean{
		exists: exists,
		value:  value,
	}
}

// Exists returns true if the value is Some, false if it is None.
func (x *Boolean) Exists() bool {
	return x.exists
}

// Value returns the []byte value. It returns nil if it is None.
func (x *Boolean) Value() bool {
	return x.value
}

// Set sets the exists and value fields.
func (x *Boolean) Set(value bool) {
	x.exists = true
	x.value = value
}

// Encode returns the SCALE encoded optional
func (x *Boolean) Encode() ([]byte, error) {
	if !x.exists {
		return []byte{0}, nil
	}
	var encodeValue []byte

	if !x.value {
		encodeValue = []byte{1}
	} else {
		encodeValue = []byte{2}
	}

	return encodeValue, nil
}

// Decode return an optional Boolean from scale encoded data
func (x *Boolean) Decode(r io.Reader) (*Boolean, error) {
	decoded, err := common.ReadByte(r)

	if err != nil {
		return nil, ErrInvalidOptional
	}

	if decoded > 2 {
		return nil, ErrInvalidOptional
	}

	if decoded == 0 {
		x.exists = false
		x.value = false
	} else if decoded == 1 {
		x.exists = true
		x.value = false
	} else {
		x.exists = true
		x.value = true
	}

	return x, nil
}

// Hash represents an optional Hash type.
type Hash struct {
	exists bool
	value  common.Hash
}

// NewHash returns a new optional.Hash
func NewHash(exists bool, value common.Hash) *Hash {
	return &Hash{
		exists: exists,
		value:  value,
	}
}

// Exists returns true if the value is Some, false if it is None.
func (x *Hash) Exists() bool {
	if x == nil {
		return false
	}
	return x.exists
}

// Value returns Hash Value
func (x *Hash) Value() common.Hash {
	if x == nil {
		return common.Hash{}
	}
	return x.value
}

// String returns the value as a string.
func (x *Hash) String() string {
	if x == nil {
		return ""
	}

	if !x.exists {
		return none
	}

	return x.value.String()
}

// Set sets the exists and value fields.
func (x *Hash) Set(exists bool, value common.Hash) {
	x.exists = exists
	x.value = value
}

// Digest is the interface implemented by the block digest
type Digest interface {
	Encode() ([]byte, error)
	Decode(io.Reader) error // Decode assumes the type byte (first byte) has been removed from the encoding.
}

// CoreHeader is a state block header
// This is copied from core/types since core/types imports this package, we cannot import core/types.
type CoreHeader struct {
	ParentHash     common.Hash `json:"parentHash"`
	Number         *big.Int    `json:"number"`
	StateRoot      common.Hash `json:"stateRoot"`
	ExtrinsicsRoot common.Hash `json:"extrinsicsRoot"`
	Digest         Digest      `json:"digest"`
}

func (h *CoreHeader) String() string {
	return fmt.Sprintf("ParentHash=%s Number=%d StateRoot=%s ExtrinsicsRoot=%s Digest=%v",
		h.ParentHash, h.Number, h.StateRoot, h.ExtrinsicsRoot, h.Digest)
}

// Header represents an optional header type
type Header struct {
	exists bool
	value  *CoreHeader
}

// NewHeader returns a new optional.Header
func NewHeader(exists bool, value *CoreHeader) *Header {
	return &Header{
		exists: exists,
		value:  value,
	}
}

// Exists returns true if the value is Some, false if it is None.
func (x *Header) Exists() bool {
	if x == nil {
		return false
	}
	return x.exists
}

// Value returns the value of the header. It returns nil if the header is None.
func (x *Header) Value() *CoreHeader {
	if x == nil {
		return nil
	}
	return x.value
}

// String returns the value as a string.
func (x *Header) String() string {
	if !x.exists || x.value == nil {
		return none
	}
	return x.value.String()
}

// Set sets the exists and value fields.
func (x *Header) Set(exists bool, value *CoreHeader) {
	x.exists = exists
	x.value = value
}

// CoreBody is the extrinsics inside a state block
type CoreBody []byte

// Body represents an optional types.Body.
type Body struct {
	exists bool
	value  CoreBody
}

// NewBody returns a new optional.Body
func NewBody(exists bool, value CoreBody) *Body {
	return &Body{
		exists: exists,
		value:  value,
	}
}

// String returns the value as a string.
func (x *Body) String() string {
	if !x.exists {
		return none
	}
	return fmt.Sprintf("%v", x.value)
}

// Set sets the exists and value fields.
func (x *Body) Set(exists bool, value CoreBody) {
	x.exists = exists
	x.value = value
}

// Value returns the value as []byte if it exists
func (x *Body) Value() []byte {
	if x == nil || !x.exists {
		return nil
	}

	return []byte(x.value)
}

// Exists returns true if the value is Some, false if it is None.
func (x *Body) Exists() bool {
	return x.exists
}
