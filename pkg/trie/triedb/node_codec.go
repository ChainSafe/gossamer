package triedb

type HashOut interface {
	comparable
	ToBytes() []byte
}
