package trie

type Trie struct {
	db *Database
	root []byte
}

func NewTrie(db *Database, root []byte) *Trie {


	return &Trie{
		db:	  db,
		root: root,
	}
}



func(t *Trie) Get(key []byte){}
func(t *Trie) Put(key []byte){}
func(t *Trie) Del(key []byte){}
