package peerid

import (
	"crypto/rand"

	"github.com/ChainSafe/gossamer/internal/client/network/types/multihash"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerID is the identifier of a peer of the network.
//
// The data is a CIDv0 compatible multihash of the protobuf encoded public key of the peer
// as specified in [specs/peer-ids](https://github.com/libp2p/specs/blob/master/peer-ids/peer-ids.md).
type PeerID struct {
	// 	multihash: Multihash,
	// multihash multihash.Multihash
	peer.ID
}

// Ed25519 will convert PeerID into ed25519 public key bytes.
func (pid PeerID) Ed25519() *[32]byte {
	pubKey, err := pid.ID.ExtractPublicKey()
	if err != nil {
		return nil
	}
	switch pubKey.(type) {
	case *crypto.Ed25519PublicKey:
	default:
		panic("huh?")
	}
	raw, err := pubKey.Raw()
	if err != nil {
		return nil
	}

	if len(raw) != 32 {
		return nil
	}

	var ret [32]byte
	copy(ret[:], raw)

	return &ret
}

// NewPeerIDFromEd25519 will create new [PeerID] from ed25519 public key bytes.
func NewPeerIDFromEd25519(b [32]byte) *PeerID {
	public, err := crypto.UnmarshalEd25519PublicKey(b[:])
	if err != nil {
		return nil
	}
	id, err := peer.IDFromPublicKey(public)
	if err != nil {
		return nil
	}
	return &PeerID{id}
}

// NewRandomPeerID will create new random [PeerID].
func NewRandomPeerID() PeerID {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	mh, err := multihash.Wrap(0x00, b)
	if err != nil {
		panic("the digest size is never too large")
	}

	id, err := peer.IDFromBytes(mh.Multihash)
	if err != nil {
		panic(err)
	}

	return PeerID{
		ID: id,
	}
}

// Create a [PeerID] parsed from bytes.
func NewPeerID(data []byte) (PeerID, error) {
	peerID, err := peer.IDFromBytes(data)
	if err != nil {
		return PeerID{}, err
	}

	return PeerID{
		peerID,
	}, nil
}
