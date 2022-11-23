package state

type KVDatabase interface {
	Get(key []byte) (value []byte, err error)
	Put(key, value []byte) (err error)
}
