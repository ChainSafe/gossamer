package parachaininteraction

// https://paritytech.github.io/polkadot/book/node/utility/availability-store.html#availability-store

type AvailabilityStore interface {
	Write(Key, Value []byte) error
	Read(Key []byte) (Value []byte, err error)
}
