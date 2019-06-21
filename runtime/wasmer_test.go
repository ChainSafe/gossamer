package runtime

import (
	"testing"
	//trie "github.com/ChainSafe/gossamer/trie"
)

// func newEmpty() *trie.Trie {
// 	db := &trie.Database{}
// 	t := trie.NewEmptyTrie(db)
// 	return t
// }

func TestExecWasmer(t *testing.T) {
	tt := newEmpty()

	ret, err := Exec(tt)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(ret)
}