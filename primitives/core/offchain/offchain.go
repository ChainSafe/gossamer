package offchain

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
