package proof

type Putter interface {
	Put(key, value []byte) (err error)
}
