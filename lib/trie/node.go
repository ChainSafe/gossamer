// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//nolint:lll
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

// Node is the interface for trie methods
type Node interface {
	EncodeAndHash() ([]byte, []byte, error)
	Decode(r io.Reader, h byte) error
	IsDirty() bool
	SetDirty(dirty bool)
	SetKey(key []byte)
	String() string
	SetEncodingAndHash([]byte, []byte)
	GetHash() []byte
	GetGeneration() uint64
	SetGeneration(uint64)
	Copy() Node
}

type (
	// Branch is a branch in the trie.
	Branch struct {
		key        []byte // partial key
		children   [16]Node
		value      []byte
		dirty      bool
		hash       []byte
		encoding   []byte
		generation uint64
		sync.RWMutex
	}

	// Leaf is a leaf in the trie.
	Leaf struct {
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

// SetGeneration sets the generation given to the branch.
func (b *Branch) SetGeneration(generation uint64) {
	b.generation = generation
}

// SetGeneration sets the generation given to the leaf.
func (l *Leaf) SetGeneration(generation uint64) {
	l.generation = generation
}

// Copy deep copies the branch.
func (b *Branch) Copy() Node {
	b.RLock()
	defer b.RUnlock()

	cpy := &Branch{
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

// Copy deep copies the leaf.
func (l *Leaf) Copy() Node {
	l.RLock()
	defer l.RUnlock()

	l.encodingMu.RLock()
	defer l.encodingMu.RUnlock()

	cpy := &Leaf{
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

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (b *Branch) SetEncodingAndHash(enc, hash []byte) {
	b.encoding = enc
	b.hash = hash
}

// SetEncodingAndHash sets the encoding and hash slices
// given to the branch. Note it does not copy them, so beware.
func (l *Leaf) SetEncodingAndHash(enc, hash []byte) {
	l.encodingMu.Lock()
	l.encoding = enc
	l.encodingMu.Unlock()

	l.hash = hash
}

// GetHash returns the hash of the branch.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (b *Branch) GetHash() []byte {
	return b.hash
}

// GetGeneration returns the generation of the branch.
func (b *Branch) GetGeneration() uint64 {
	return b.generation
}

// GetGeneration returns the generation of the leaf.
func (l *Leaf) GetGeneration() uint64 {
	return l.generation
}

// GetHash returns the hash of the leaf.
// Note it does not copy it, so modifying
// the returned hash will modify the hash
// of the branch.
func (l *Leaf) GetHash() []byte {
	return l.hash
}

func (b *Branch) String() string {
	if len(b.value) > 1024 {
		return fmt.Sprintf(
			"branch key=%x childrenBitmap=%16b value (hashed)=%x dirty=%v",
			b.key, b.childrenBitmap(), common.MustBlake2bHash(b.value), b.dirty)
	}
	return fmt.Sprintf("branch key=%x childrenBitmap=%16b value=%v dirty=%v", b.key, b.childrenBitmap(), b.value, b.dirty)
}

func (l *Leaf) String() string {
	if len(l.value) > 1024 {
		return fmt.Sprintf("leaf key=%x value (hashed)=%x dirty=%v", l.key, common.MustBlake2bHash(l.value), l.dirty)
	}
	return fmt.Sprintf("leaf key=%x value=%v dirty=%v", l.key, l.value, l.dirty)
}

func (b *Branch) childrenBitmap() uint16 {
	var bitmap uint16
	var i uint
	for i = 0; i < 16; i++ {
		if b.children[i] != nil {
			bitmap = bitmap | 1<<i
		}
	}
	return bitmap
}

func (b *Branch) numChildren() int {
	var i, count int
	for i = 0; i < 16; i++ {
		if b.children[i] != nil {
			count++
		}
	}
	return count
}

// IsDirty returns the dirty status of the leaf.
func (l *Leaf) IsDirty() bool {
	return l.dirty
}

// IsDirty returns the dirty status of the branch.
func (b *Branch) IsDirty() bool {
	return b.dirty
}

// SetDirty sets the dirty status to the leaf.
func (l *Leaf) SetDirty(dirty bool) {
	l.dirty = dirty
}

// SetDirty sets the dirty status to the branch.
func (b *Branch) SetDirty(dirty bool) {
	b.dirty = dirty
}

// SetKey sets the key to the leaf.
// Note it does not copy it so modifying the passed key
// will modify the key stored in the leaf.
func (l *Leaf) SetKey(key []byte) {
	l.key = key
}

// SetKey sets the key to the branch.
// Note it does not copy it so modifying the passed key
// will modify the key stored in the branch.
func (b *Branch) SetKey(key []byte) {
	b.key = key
}

// EncodeAndHash returns the encoding of the branch and
// the blake2b hash digest of the encoding of the branch.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (b *Branch) EncodeAndHash() (encoding, hash []byte, err error) {
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

// EncodeAndHash returns the encoding of the leaf and
// the blake2b hash digest of the encoding of the leaf.
// If the encoding is less than 32 bytes, the hash returned
// is the encoding and not the hash of the encoding.
func (l *Leaf) EncodeAndHash() (encoding, hash []byte, err error) {
	l.encodingMu.RLock()
	if !l.IsDirty() && l.encoding != nil && l.hash != nil {
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

func decodeBytes(in []byte) (Node, error) {
	buffer := bytes.NewBuffer(in)
	return decode(buffer)
}

// decode wraps the decoding of different node types back into a node
func decode(r io.Reader) (Node, error) {
	header, err := readByte(r)
	if err != nil {
		return nil, err
	}

	nodeType := header >> 6
	if nodeType == 1 {
		l := new(Leaf)
		err := l.Decode(r, header)
		return l, err
	} else if nodeType == 2 || nodeType == 3 {
		b := new(Branch)
		err := b.Decode(r, header)
		return b, err
	}

	return nil, errors.New("cannot decode invalid encoding into node")
}

// Decode decodes a byte array with the encoding specified at the top of this package into a branch node
// Note that since the encoded branch stores the hash of the children nodes, we aren't able to reconstruct the child
// nodes from the encoding. This function instead stubs where the children are known to be with an empty leaf.
func (b *Branch) Decode(r io.Reader, header byte) (err error) {
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

			b.children[i] = &Leaf{
				hash: hash,
			}
		}
	}

	b.dirty = true

	return nil
}

// Decode decodes a byte array with the encoding specified at the top of this package into a leaf node
func (l *Leaf) Decode(r io.Reader, header byte) (err error) {
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

func (b *Branch) header() ([]byte, error) {
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

func (l *Leaf) header() ([]byte, error) {
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
