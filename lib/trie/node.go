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

// Modified Merkle-Patricia Trie
// See https://github.com/w3f/polkadot-spec/blob/master/runtime-environment-spec/polkadot_re_spec.pdf for the full specification.
//
// Note that for the following definitions, `|` denotes concatenation
//
// Branch encoding:
// NodeHeader | Extra partial key length | Partial Key | Value
// `NodeHeader` is a byte such that:
// most significant two bits of `NodeHeader`: 10 if branch w/o value, 11 if branch w/ value
// least significant six bits of `NodeHeader`: if len(key) > 62, 0x3f, otherwise len(key)
// `Extra partial key length` is included if len(key) > 63 and consists of the remaining key length
// `Partial Key` is the branch's key
// `Value` is: Children Bitmap | SCALE Branch node Value | Hash(Enc(Child[i_1])) | Hash(Enc(Child[i_2])) | ... | Hash(Enc(Child[i_n]))
//
// Leaf encoding:
// NodeHeader | Extra partial key length | Partial Key | Value
// `NodeHeader` is a byte such that:
// most significant two bits of `NodeHeader`: 01
// least significant six bits of `NodeHeader`: if len(key) > 62, 0x3f, otherwise len(key)
// `Extra partial key length` is included if len(key) > 63 and consists of the remaining key length
// `Partial Key` is the leaf's key
// `Value` is the leaf's SCALE encoded value

package trie

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// node is the interface for trie methods
type node interface {
	encodeAndHash() ([]byte, []byte, error)
	encode() ([]byte, error)
	decode(r io.Reader, h byte) error
	isDirty() bool
	setDirty(dirty bool)
	setKey(key []byte)
	String() string
	setEncodingAndHash([]byte, []byte)
	getHash() []byte
	getGeneration() uint64
}

type (
	branch struct {
		key        []byte // partial key
		children   [16]node
		value      []byte
		dirty      bool
		hash       []byte
		encoding   []byte
		generation uint64
		sync.RWMutex
	}
	leaf struct {
		key        []byte // partial key
		value      []byte
		dirty      bool
		hash       []byte
		encoding   []byte
		generation uint64
		sync.RWMutex
	}
)

func (b *branch) copy() *branch {
	cpy := &branch{
		key:        make([]byte, len(b.key)),
		children:   b.children,
		value:      make([]byte, len(b.value)),
		dirty:      b.dirty,
		hash:       make([]byte, len(b.hash)),
		encoding:   make([]byte, len(b.encoding)),
		generation: b.generation,
	}
	copy(cpy.key, b.key)
	copy(cpy.value, b.value)
	copy(cpy.hash, b.hash)
	copy(cpy.encoding, b.encoding)
	return cpy
}

func (l *leaf) copy() *leaf {
	cpy := &leaf{
		key:        make([]byte, len(l.key)),
		value:      make([]byte, len(l.value)),
		dirty:      l.dirty,
		hash:       make([]byte, len(l.hash)),
		encoding:   make([]byte, len(l.encoding)),
		generation: l.generation,
	}
	copy(cpy.key, l.key)
	copy(cpy.value, l.value)
	copy(cpy.hash, l.hash)
	copy(cpy.encoding, l.encoding)
	return cpy
}

func (b *branch) setEncodingAndHash(enc, hash []byte) {
	b.encoding = enc
	b.hash = hash
}

func (l *leaf) setEncodingAndHash(enc, hash []byte) {
	l.encoding = enc
	l.hash = hash
}

func (b *branch) getHash() []byte {
	return b.hash
}

func (b *branch) getGeneration() uint64 {
	return b.generation
}

func (l *leaf) getGeneration() uint64 {
	return l.generation
}

func (l *leaf) getHash() []byte {
	return l.hash
}

func (b *branch) String() string {
	if len(b.value) > 1024 {
		return fmt.Sprintf("branch key=%x childrenBitmap=%16b value (hashed)=%x dirty=%v", b.key, b.childrenBitmap(), common.MustBlake2bHash(b.value), b.dirty)
	}
	return fmt.Sprintf("branch key=%x childrenBitmap=%16b value=%x dirty=%v", b.key, b.childrenBitmap(), b.value, b.dirty)
}

func (l *leaf) String() string {
	if len(l.value) > 1024 {
		return fmt.Sprintf("leaf key=%x value (hashed)=%x dirty=%v", l.key, common.MustBlake2bHash(l.value), l.dirty)
	}
	return fmt.Sprintf("leaf key=%x value=%x dirty=%v", l.key, l.value, l.dirty)
}

func (b *branch) childrenBitmap() uint16 {
	var bitmap uint16
	var i uint
	for i = 0; i < 16; i++ {
		if b.children[i] != nil {
			bitmap = bitmap | 1<<i
		}
	}
	return bitmap
}

func (b *branch) numChildren() int {
	var i, count int
	for i = 0; i < 16; i++ {
		if b.children[i] != nil {
			count++
		}
	}
	return count
}

func (l *leaf) isDirty() bool {
	return l.dirty
}

func (b *branch) isDirty() bool {
	return b.dirty
}

func (l *leaf) setDirty(dirty bool) {
	l.dirty = dirty
}

func (b *branch) setDirty(dirty bool) {
	b.dirty = dirty
}

func (l *leaf) setKey(key []byte) {
	l.key = key
}

func (b *branch) setKey(key []byte) {
	b.key = key
}

// Encode is the high-level function wrapping the encoding for different node types
// encoding has the following format:
// NodeHeader | Extra partial key length | Partial Key | Value
func encode(n node) ([]byte, error) {
	switch n := n.(type) {
	case *branch:
		return n.encode()
	case *leaf:
		return n.encode()
	case nil:
		return []byte{0}, nil
	}

	return nil, nil
}

func (b *branch) encodeAndHash() ([]byte, []byte, error) {
	if !b.dirty && b.encoding != nil && b.hash != nil {
		return b.encoding, b.hash, nil
	}

	enc, err := b.encode()
	if err != nil {
		return nil, nil, err
	}

	if len(enc) < 32 {
		b.encoding = enc
		b.hash = enc
		return enc, enc, nil
	}

	hash, err := common.Blake2bHash(enc)
	if err != nil {
		return nil, nil, err
	}

	b.encoding = enc
	b.hash = hash[:]
	return enc, hash[:], nil
}

// Encode encodes a branch with the encoding specified at the top of this package
func (b *branch) encode() ([]byte, error) {
	if !b.dirty && b.encoding != nil {
		return b.encoding, nil
	}

	encoding, err := b.header()
	if err != nil {
		return nil, err
	}

	encoding = append(encoding, nibblesToKeyLE(b.key)...)
	encoding = append(encoding, common.Uint16ToBytes(b.childrenBitmap())...)

	if b.value != nil {
		buffer := bytes.Buffer{}
		se := scale.Encoder{Writer: &buffer}
		_, err = se.Encode(b.value)
		if err != nil {
			return encoding, err
		}
		encoding = append(encoding, buffer.Bytes()...)
	}

	for _, child := range b.children {
		if child != nil {
			hasher, err := NewHasher()
			if err != nil {
				return nil, err
			}

			encChild, err := hasher.Hash(child)
			if err != nil {
				return encoding, err
			}

			scEncChild, err := scale.Encode(encChild)
			if err != nil {
				return encoding, err
			}
			encoding = append(encoding, scEncChild[:]...)
		}
	}

	return encoding, nil
}

func (l *leaf) encodeAndHash() ([]byte, []byte, error) {
	if !l.isDirty() && l.encoding != nil && l.hash != nil {
		return l.encoding, l.hash, nil
	}

	enc, err := l.encode()
	if err != nil {
		return nil, nil, err
	}

	if len(enc) < 32 {
		l.encoding = enc
		l.hash = enc
		return enc, enc, nil
	}

	hash, err := common.Blake2bHash(enc)
	if err != nil {
		return nil, nil, err
	}

	l.encoding = enc
	l.hash = hash[:]
	return enc, hash[:], nil
}

// Encode encodes a leaf with the encoding specified at the top of this package
func (l *leaf) encode() ([]byte, error) {
	if !l.dirty && l.encoding != nil {
		return l.encoding, nil
	}

	encoding, err := l.header()
	if err != nil {
		return nil, err
	}

	encoding = append(encoding, nibblesToKeyLE(l.key)...)

	buffer := bytes.Buffer{}
	se := scale.Encoder{Writer: &buffer}
	_, err = se.Encode(l.value)
	if err != nil {
		return encoding, err
	}
	encoding = append(encoding, buffer.Bytes()...)
	l.encoding = encoding
	return encoding, nil
}

func decodeBytes(in []byte) (node, error) {
	r := &bytes.Buffer{}
	_, err := r.Write(in)
	if err != nil {
		return nil, err
	}

	return decode(r)
}

// Decode wraps the decoding of different node types back into a node
func decode(r io.Reader) (node, error) {
	header, err := readByte(r)
	if err != nil {
		return nil, err
	}

	nodeType := header >> 6
	if nodeType == 1 {
		l := new(leaf)
		err := l.decode(r, header)
		return l, err
	} else if nodeType == 2 || nodeType == 3 {
		b := new(branch)
		err := b.decode(r, header)
		return b, err
	}

	return nil, errors.New("cannot decode invalid encoding into node")
}

// Decode decodes a byte array with the encoding specified at the top of this package into a branch node
// Note that since the encoded branch stores the hash of the children nodes, we aren't able to reconstruct the child
// nodes from the encoding. This function instead stubs where the children are known to be with an empty leaf.
func (b *branch) decode(r io.Reader, header byte) (err error) {
	if header == 0 {
		header, err = readByte(r)
		if err != nil {
			return err
		}
	}

	nodeType := header >> 6
	if nodeType != 2 && nodeType != 3 {
		return fmt.Errorf("cannot decode node to branch")
	}

	keyLen := header & 0x3f
	b.key, err = decodeKey(r, keyLen)
	if err != nil {
		return err
	}

	childrenBitmap := make([]byte, 2)
	_, err = r.Read(childrenBitmap)
	if err != nil {
		return err
	}

	sd := &scale.Decoder{Reader: r}

	if nodeType == 3 {
		// branch w/ value
		value, err := sd.Decode([]byte{})
		if err != nil {
			return err
		}
		b.value = value.([]byte)
	}

	for i := 0; i < 16; i++ {
		if (childrenBitmap[i/8]>>(i%8))&1 == 1 {
			hash, err := sd.Decode([]byte{})
			if err != nil {
				return err
			}

			b.children[i] = &leaf{
				hash: hash.([]byte),
			}
		}
	}

	b.dirty = true

	return nil
}

// Decode decodes a byte array with the encoding specified at the top of this package into a leaf node
func (l *leaf) decode(r io.Reader, header byte) (err error) {
	if header == 0 {
		header, err = readByte(r)
		if err != nil {
			return err
		}
	}

	nodeType := header >> 6
	if nodeType != 1 {
		return fmt.Errorf("cannot decode node to leaf")
	}

	keyLen := header & 0x3f
	l.key, err = decodeKey(r, keyLen)
	if err != nil {
		return err
	}

	sd := &scale.Decoder{Reader: r}
	value, err := sd.Decode([]byte{})
	if err != nil {
		return err
	}

	if len(value.([]byte)) > 0 {
		l.value = value.([]byte)
	}

	l.dirty = true

	return nil
}

func (b *branch) header() ([]byte, error) {
	var header byte
	if b.value == nil {
		header = 2 << 6
	} else {
		header = 3 << 6
	}
	var encodePkLen []byte
	var err error

	if len(b.key) >= 63 {
		header = header | 0x3f
		encodePkLen, err = encodeExtraPartialKeyLength(len(b.key))
		if err != nil {
			return nil, err
		}
	} else {
		header = header | byte(len(b.key))
	}

	fullHeader := append([]byte{header}, encodePkLen...)
	return fullHeader, nil
}

func (l *leaf) header() ([]byte, error) {
	var header byte = 1 << 6
	var encodePkLen []byte
	var err error

	if len(l.key) >= 63 {
		header = header | 0x3f
		encodePkLen, err = encodeExtraPartialKeyLength(len(l.key))
		if err != nil {
			return nil, err
		}
	} else {
		header = header | byte(len(l.key))
	}

	fullHeader := append([]byte{header}, encodePkLen...)
	return fullHeader, nil
}

func encodeExtraPartialKeyLength(pkLen int) ([]byte, error) {
	pkLen -= 63
	fullHeader := []byte{}

	if pkLen >= 1<<16 {
		return nil, errors.New("partial key length greater than or equal to 2^16")
	}

	for i := 0; i < 1<<16; i++ {
		if pkLen < 255 {
			fullHeader = append(fullHeader, byte(pkLen))
			break
		} else {
			fullHeader = append(fullHeader, byte(255))
			pkLen -= 255
		}
	}

	return fullHeader, nil
}

func decodeKey(r io.Reader, keyLen byte) ([]byte, error) {
	var totalKeyLen int = int(keyLen)

	if keyLen == 0x3f {
		// partial key longer than 63, read next bytes for rest of pk len
		for {
			nextKeyLen, err := readByte(r)
			if err != nil {
				return nil, err
			}
			totalKeyLen += int(nextKeyLen)

			if nextKeyLen < 0xff {
				break
			}

			if totalKeyLen >= 1<<16 {
				return nil, errors.New("partial key length greater than or equal to 2^16")
			}
		}
	}

	if totalKeyLen != 0 {
		key := make([]byte, totalKeyLen/2+totalKeyLen%2)
		_, err := r.Read(key)
		if err != nil {
			return key, err
		}

		return keyToNibbles(key)[totalKeyLen%2:], nil
	}

	return []byte{}, nil
}

func readByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)
	_, err := r.Read(buf)
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}
