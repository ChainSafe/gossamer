package cache

type TrieCache interface {
	GetValue(key []byte) []byte
	SetValue(key []byte, value []byte)
}
