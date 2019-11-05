package crypto

type Keypair interface {
	Sign(msg []byte) []byte 
	Public() *PublicKey
	Private() *PrivateKey
}

type PublicKey interface {
	Verify(msg, sig []byte) bool
}

type PrivateKey interface {
	Sign(msg []byte) []byte
	Public () *PublicKey
}