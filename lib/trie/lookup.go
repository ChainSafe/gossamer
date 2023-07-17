package trie

type Lookup struct {
	db   HashDB
	hash []byte
	//TODO: implement cache and recorder
}

func NewLookup(db HashDB, hash []byte) *Lookup {
	return &Lookup{db, hash}
}

func (l Lookup) Lookup(key []byte, nibbleKey *NibbleSlice) (*[]byte, error) {
	return l.lookupWithoutCache(nibbleKey, key)
}

func (l Lookup) lookupWithoutCache(nibbleKey *NibbleSlice, fullKey []byte) (*[]byte, error) {
	//TODO: finish it
	return nil, nil
}
