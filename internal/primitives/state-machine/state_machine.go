package statemachine

// / Multiple key value state.
// / States are ordered by root storage key.
// #[derive(PartialEq, Eq, Clone)]
// pub struct KeyValueStates(pub Vec<KeyValueStorageLevel>);
type KeyValueStates []KeyValueStorageLevel

// / A key value state at any storage level.
// #[derive(PartialEq, Eq, Clone)]
// pub struct KeyValueStorageLevel {
type KeyValueStorageLevel struct {
	/// State root of the level, for
	/// top trie it is as an empty byte array.
	// pub state_root: Vec<u8>,
	StateRoot []byte
	/// Storage of parents, empty for top root or
	/// when exporting (building proof).
	// pub parent_storage_keys: Vec<Vec<u8>>,
	ParentStorageKeys [][]byte
	/// Pair of key and values from this state.
	// pub key_values: Vec<(Vec<u8>, Vec<u8>)>,
	KeyValues []struct {
		Key   []byte
		Value []byte
	}
}
