package trie

type Trie struct {
	db *Database
	root [32]byte
}

func NewTrie(db *Database, root [32]byte) *Trie {


	return &Trie{
		db:	  db,
		root: root,
	}
}

func(t *Trie) Put(k, v  []byte){}

func(t *Trie) Get(key []byte){}

func(t *Trie) Del(key []byte){}
