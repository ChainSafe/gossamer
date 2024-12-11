package keys

import (
	"strings"
)

// / List of all well known keys and prefixes in storage.
var (
	// / Prefix of the default child storage keys in the top trie.
	DefaultChildStorageKeyPrefix = []byte(":child_storage:default:")
)

// / Whether a key is a child storage key.
// /
// / This is convenience function which basically checks if the given `key` starts
// / with `CHILD_STORAGE_KEY_PREFIX` and doesn't do anything apart from that.
func IsChildStorageKey(key []byte) bool {
	i := strings.Index(string(key), string(DefaultChildStorageKeyPrefix))
	return i == 0
}
