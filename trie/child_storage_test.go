package trie

import (
	"reflect"
	"testing"
)

func TestPutAndGetChild(t *testing.T) {
	childKey := []byte("default")
	childTrie := buildSmallTrie(t)
	parentTrie := NewEmptyTrie(nil)

	err := parentTrie.PutChild(childKey, childTrie)
	if err != nil {
		t.Fatal(err)
	}

	childTrieRes, err := parentTrie.GetChild(childKey)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(childTrie, childTrieRes) {
		t.Fatalf("Fail: got %v expected %v", childTrieRes, childTrie)
	}
}