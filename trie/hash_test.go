package trie

import (
	"bytes"
	"math/rand"
	"testing"

	scale "github.com/ChainSafe/gossamer/codec"
	hexcodec "github.com/ChainSafe/gossamer/common/codec"
)

func generateRandBytes(size int) []byte {
	r := *rand.New(rand.NewSource(rand.Int63()))
	buf := make([]byte, r.Intn(size)+1)
	r.Read(buf)
	return buf
}

func generateRand(size int) [][]byte {
	rt := make([][]byte, size)
	r := *rand.New(rand.NewSource(rand.Int63()))
	for i := range rt {
		buf := make([]byte, r.Intn(379)+1)
		r.Read(buf)
		rt[i] = buf
	}
	return rt
}

func TestNewHasher(t *testing.T) {
	hasher, err := newHasher()
	if err != nil {
		t.Fatalf("error creating new hasher: %s", err)
	} else if hasher == nil {
		t.Fatal("did not create new hasher")
	}

	_, err = hasher.hash.Write([]byte("noot"))
	if err != nil {
		t.Error(err)
	}

	sum := hasher.hash.Sum(nil)
	if sum == nil {
		t.Error("did not sum hash")
	}

	hasher.hash.Reset()
}

func TestEncodeLen(t *testing.T) {
	tests := []struct {
		input    node
		expected []byte
	}{
		{&extension{key: []byte{0x00}}, []byte{128, 1}},
		{&extension{key: []byte{0x00, 0x01, 0x02, 0x03}}, []byte{128, 4}},
		{&leaf{key: []byte{0x00}}, []byte{1, 1}},
		{&leaf{key: []byte{0x00, 0x01, 0x02, 0x03}}, []byte{1, 4}},
	}

	for _, test := range tests {
		res, err := encodeLen(test.input)
		if !bytes.Equal(res, test.expected) {
			t.Errorf("Fail when encoding node length: got %x expected %x", res, test.expected)
		} else if err != nil {
			t.Errorf("Fail when encoding node length: %s", err)
		}
	}

	_, err := encodeLen(&branch{})
	if err == nil {
		t.Errorf("did not error when encoding len of branch node")
	}
}

func TestEncodeLenExtensions(t *testing.T) {
	randKeys := generateRand(100)
	for _, testKey := range randKeys {
		n := &extension{key: testKey}
		var expected []byte
		if len(testKey) >= 125 {
			expected = []byte{128, 127, byte(len(testKey) - 125)}
		} else {
			expected = []byte{128, byte(len(testKey))}
		}

		res, err := encodeLen(n)
		if !bytes.Equal(res, expected) {
			t.Errorf("Fail when encoding node length: got %x expected %x", res, expected)
		} else if err != nil {
			t.Errorf("Fail when encoding node length: %s", err)
		}
	}
}

func TestEncodeLenLeaves(t *testing.T) {
	randKeys := generateRand(100)
	for _, testKey := range randKeys {
		n := &leaf{key: testKey}
		var expected []byte
		if len(testKey) >= 126 {
			expected = []byte{1, 127, byte(len(testKey) - 126)}
		} else {
			expected = []byte{1, byte(len(testKey))}
		}

		res, err := encodeLen(n)
		if !bytes.Equal(res, expected) {
			t.Errorf("Fail when encoding node length: got %x expected %x", res, expected)
		} else if err != nil {
			t.Errorf("Fail when encoding node length: %s", err)
		}
	}
}

func TestEncodeLeaves(t *testing.T) {
	randKeys := generateRand(100)
	randVals := generateRand(100)

	for i, testKey := range randKeys {
		n := &leaf{key: testKey, value: randVals[i]}
		var expected []byte
		if len(testKey) >= 126 {
			expected = []byte{1, 127, byte(len(testKey) - 126)}
		} else {
			expected = []byte{1, byte(len(testKey))}
		}

		encHex := keyToHex(n.key)
		encHex = hexcodec.Encode(encHex[0 : len(encHex)-1])
		expected = append(expected, encHex...)

		buf := bytes.Buffer{}
		encoder := &scale.Encoder{&buf}
		_, err := encoder.Encode(n.value)
		if err != nil {
			t.Fatalf("Fail when encoding value with scale: %s", err)
		}

		expected = append(expected, buf.Bytes()...)

		res, err := n.Encode()
		if !bytes.Equal(res, expected) {
			t.Errorf("Fail when encoding node length: got %x expected %x", res, expected)
		} else if err != nil {
			t.Errorf("Fail when encoding node length: %s", err)
		}
	}
}

func TestHashLeaf(t *testing.T) {
	n := &leaf{key: generateRandBytes(380), value: generateRandBytes(64)}
	h, err := n.Hash()
	if err != nil {
		t.Errorf("did not hash leaf node: %s", err)
	} else if h == nil {
		t.Errorf("did not hash leaf node: nil")
	}
}
