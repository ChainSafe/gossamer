package trie

import (
	"math/rand"
	"testing"
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

func TestHashLeaf(t *testing.T) {
	n := &leaf{key: generateRandBytes(380), value: generateRandBytes(64)}
	h, err := Hash(n)
	if err != nil {
		t.Errorf("did not hash leaf node: %s", err)
	} else if h == nil {
		t.Errorf("did not hash leaf node: nil")
	}
}

func TestHashBranch(t *testing.T) {
	n := &branch{}
	n.children[3] = &leaf{key: generateRandBytes(380), value: generateRandBytes(380)}
	h, err := Hash(n)
	if err != nil {
		t.Errorf("did not hash branch node: %s", err)
	} else if h == nil {
		t.Errorf("did not hash branch node: nil")
	}
}