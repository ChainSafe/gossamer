package triedb

type RecordedForKey uint8

const (
	RecordedForKeyValue RecordedForKey = iota
	RecordedForKeyHash
	RecordedForKeyNone
)

type TrieRecorder[Out HashOut] interface {
	record(access TrieAccess[Out])
	trieNodesRecordedForKey(key []byte) RecordedForKey
}

// TrieAccess is used to report the trie access to the TrieRecorder
type TrieAccess[Out HashOut] interface {
	Type() string
}

type (
	// TrieAccessNode means that the given node was accessed using its hash
	TrieAccessNode[H HashOut] struct {
		hash H
		node Node[H]
	}

	// TrieAccessEncodedNode means that the given encodedNode was accessed using its hash
	TrieAccessEncodedNode[H HashOut] struct {
		hash        H
		encodedNode []byte
	}

	// TrieAccessValue means that the given value was accessed using its hash
	// fullKey is the key to access this value in the trie
	// Should map to RecordedForKeyValue when checking the recorder
	TrieAccessValue[H HashOut] struct {
		hash    H
		value   []byte
		fullKey []byte
	}

	// TrieAccessInlineValue means that a value stored in an inlined node was accessed
	// The given fullKey is the key to access this value in the trie
	// Should map to RecordedForKeyValue when checking the recorder
	TrieAccessInlineValue[H HashOut] struct {
		fullKey []byte
	}

	// TrieAccessHash means that the hash of the value for a given fullKey was accessed
	// Should map to RecordedForKeyHash when checking the recorder
	TrieAccessHash[H HashOut] struct {
		fullKey []byte
	}

	// TrieAccessNotExisting means that the value/hash for fullKey was accessed, but it couldn't be found in the trie
	// Should map to RecordedForKeyValue when checking the recorder
	TrieAccessNotExisting struct {
		fullKey []byte
	}
)

func (a TrieAccessNode[H]) Type() string        { return "Node" }
func (a TrieAccessEncodedNode[H]) Type() string { return "EncodedNode" }
func (a TrieAccessValue[H]) Type() string       { return "Value" }
func (a TrieAccessInlineValue[H]) Type() string { return "InlineValue" }
func (a TrieAccessHash[H]) Type() string        { return "Hash" }
func (a TrieAccessNotExisting) Type() string    { return "NotExisting" }
