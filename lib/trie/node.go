// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

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
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// node is the interface for trie methods
type node interface {
	encodeAndHash() ([]byte, []byte, error)
	decode(r io.Reader, h byte) error
	isDirty() bool
	setDirty(dirty bool)
	setKey(key []byte)
	String() string
	setEncodingAndHash([]byte, []byte)
	getHash() []byte
	getGeneration() uint64
	setGeneration(uint64)
	copy() node
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
		encodingMu sync.RWMutex
		generation uint64
		sync.RWMutex
	}
)

func (b *branch) setGeneration(generation uint64) {
	b.generation = generation
}

func (l *leaf) setGeneration(generation uint64) {
	l.generation = generation
}

func (b *branch) copy() node {
	b.RLock()
	defer b.RUnlock()

	cpy := &branch{
		key:        make([]byte, len(b.key)),
		children:   b.children, // copy interface pointers
		value:      nil,
		dirty:      b.dirty,
		hash:       make([]byte, len(b.hash)),
		encoding:   make([]byte, len(b.encoding)),
		generation: b.generation,
	}
	copy(cpy.key, b.key)

	// nil and []byte{} are encoded differently, watch out!
	if b.value != nil {
		cpy.value = make([]byte, len(b.value))
		copy(cpy.value, b.value)
	}

	copy(cpy.hash, b.hash)
	copy(cpy.encoding, b.encoding)
	return cpy
}

func (l *leaf) copy() node {
	l.RLock()
	defer l.RUnlock()

	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()

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
	l.encodingMu.Lock()
	l.encoding = enc
	l.encodingMu.Unlock()

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
	return fmt.Sprintf("branch key=%x childrenBitmap=%16b value=%v dirty=%v", b.key, b.childrenBitmap(), b.value, b.dirty)
}

func (l *leaf) String() string {
	if len(l.value) > 1024 {
		return fmt.Sprintf("leaf key=%x value (hashed)=%x dirty=%v", l.key, common.MustBlake2bHash(l.value), l.dirty)
	}
	return fmt.Sprintf("leaf key=%x value=%v dirty=%v", l.key, l.value, l.dirty)
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

func (b *branch) encodeAndHash() (encoding, hash []byte, err error) {
	if !b.dirty && b.encoding != nil && b.hash != nil {
		return b.encoding, b.hash, nil
	}

	buffer := encodingBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer encodingBufferPool.Put(buffer)

	err = encodeBranch(b, buffer, false)
	if err != nil {
		return nil, nil, err
	}

	bufferBytes := buffer.Bytes()

	b.encoding = make([]byte, len(bufferBytes))
	copy(b.encoding, bufferBytes)
	encoding = b.encoding // no need to copy

	if buffer.Len() < 32 {
		b.hash = make([]byte, len(bufferBytes))
		copy(b.hash, bufferBytes)
		hash = b.hash // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}
	b.hash = hashArray[:]
	hash = b.hash // no need to copy

	return encoding, hash, nil
}

func (l *leaf) encodeAndHash() (encoding, hash []byte, err error) {
	l.encodingMu.RLock()
	if !l.isDirty() && l.encoding != nil && l.hash != nil {
		l.encodingMu.RUnlock()
		return l.encoding, l.hash, nil
	}
	l.encodingMu.RUnlock()

	buffer := encodingBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer encodingBufferPool.Put(buffer)

	err = encodeLeaf(l, buffer)
	if err != nil {
		return nil, nil, err
	}

	bufferBytes := buffer.Bytes()

	l.encodingMu.Lock()
	// TODO remove this copying since it defeats the purpose of `buffer`
	// and the sync.Pool.
	l.encoding = make([]byte, len(bufferBytes))
	copy(l.encoding, bufferBytes)
	l.encodingMu.Unlock()
	encoding = l.encoding // no need to copy

	if len(bufferBytes) < 32 {
		l.hash = make([]byte, len(bufferBytes))
		copy(l.hash, bufferBytes)
		hash = l.hash // no need to copy
		return encoding, hash, nil
	}

	// Note: using the sync.Pool's buffer is useful here.
	hashArray, err := common.Blake2bHash(buffer.Bytes())
	if err != nil {
		return nil, nil, err
	}

	l.hash = hashArray[:]
	hash = l.hash // no need to copy

	return encoding, hash, nil
}

func decodeBytes(in []byte) (node, error) {
	buffer := bytes.NewBuffer(in)
	return decode(buffer)
}

// decode wraps the decoding of different node types back into a node
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

	sd := scale.NewDecoder(r)

	if nodeType == 3 {
		var value []byte
		// branch w/ value
		err := sd.Decode(&value)
		if err != nil {
			return err
		}
		b.value = value
	}

	for i := 0; i < 16; i++ {
		if (childrenBitmap[i/8]>>(i%8))&1 == 1 {
			var hash []byte
			err := sd.Decode(&hash)
			if err != nil {
				return err
			}

			b.children[i] = &leaf{
				hash: hash,
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

	sd := scale.NewDecoder(r)
	var value []byte
	err = sd.Decode(&value)
	if err != nil {
		return err
	}

	if len(value) > 0 {
		l.value = value
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

var ErrPartialKeyTooBig = errors.New("partial key length greater than or equal to 2^16")

func encodeExtraPartialKeyLength(pkLen int) ([]byte, error) {
	pkLen -= 63
	fullHeader := []byte{}

	if pkLen >= 1<<16 {
		return nil, ErrPartialKeyTooBig
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
	var totalKeyLen = int(keyLen)

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
