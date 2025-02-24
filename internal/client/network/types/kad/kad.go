package kad

import (
	"time"

	peerid "github.com/ChainSafe/gossamer/internal/client/network/types/peer-id"
)

// Key is the (opaque) key of a record.
type Key []byte

// Record is a record stored in the DHT.
type Record struct {
	// Key of the record.
	Key Key
	// Value of the record.
	Value []byte
	// The (original) publisher of the record.
	Publisher *peerid.PeerID
	// The expiration time as measured by a local, monotonic clock.
	Expires *time.Time
}

// PeerRecord is a record either received by the given peer or retrieved from the local record store.
type PeerRecord struct {
	// The peer from whom the record was received. nil if the record was retrieved from local storage.
	Peer   *peerid.PeerID
	Record Record
}
