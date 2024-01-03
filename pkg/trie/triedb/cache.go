package triedb

// CachedValue a value as cached by TrieCache
type CachedValue[H comparable] interface {
	Type() string
}
type (
	// The value doesn't exists in ithe trie
	NonExisting struct{}
	// We cached the hash, because we did not yet accessed the data
	ExistingHash[H comparable] struct {
		hash H
	}
	// The value xists in the trie
	Existing[H comparable] struct {
		hash H      // The hash of the value
		data []byte // The actual data of the value
	}
)

func (v NonExisting) Type() string     { return "NonExisting" }
func (v ExistingHash[H]) Type() string { return "ExistingHash" }
func (v Existing[H]) Type() string     { return "Existing" }

type TrieCache[Out HashOut] interface {
	LookupValueForKey(key []byte) *CachedValue[Out]
	CacheValueForKey(key []byte, value CachedValue[Out])
	GetOrInsertNode(hash Out, fetchNode func() (Node[Out], error))
	GetNode(hash Out) Node[Out]
}
