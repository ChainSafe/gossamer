package messages

// P2PMessage must be implemented by all network messages
type P2PMessage interface {
	String() string
	Encode() ([]byte, error)
	Decode(in []byte) (err error)
}
