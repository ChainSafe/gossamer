package offchain

// / Offchain DB persistent (non-fork-aware) storage.
// pub trait OffchainStorage: Clone + Send + Sync {
type OffchainStorage interface {
	/// Persist a value in storage under given key and prefix.
	// 	fn set(&mut self, prefix: &[u8], key: &[u8], value: &[u8]);
	Set(prefix, key, value []byte)

	/// Clear a storage entry under given key and prefix.
	// 	fn remove(&mut self, prefix: &[u8], key: &[u8]);
	Remove(prefix, key []byte)

	/// Retrieve a value from storage under given key and prefix.
	// 	fn get(&self, prefix: &[u8], key: &[u8]) -> Option<Vec<u8>>;
	Get(prefix, key []byte) []byte

	/// Replace the value in storage if given old_value matches the current one.
	///
	/// Returns `true` if the value has been set and false otherwise.
	//		fn compare_and_set(
	//			&mut self,
	//			prefix: &[u8],
	//			key: &[u8],
	//			old_value: Option<&[u8]>,
	//			new_value: &[u8],
	//		) -> bool;
	//	}
	CompareAndSet(prefix, key, oldValue, newValue []byte) bool
}

// / Change to be applied to the offchain worker db in regards to a key.
type OffchainOverlayedChanges interface {
	OffchainOverlayedChangeRemove | OffchainOverlayedChangeSetValue
}

// / Change to be applied to the offchain worker db in regards to a key.
type OffchainOverlayedChange any

// / Remove the data associated with the key
type OffchainOverlayedChangeRemove struct{}

// / Overwrite the value of an associated key
type OffchainOverlayedChangeSetValue []byte
