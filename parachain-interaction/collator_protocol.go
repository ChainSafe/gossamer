package parachaininteraction

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

/* HOW
Look into smoldot https://github.com/smol-dot/smoldot

Validators:
- be ready to accept connections from collators. Have open peers slots for adding new connections from collators
- (after connection) collators with declare their `public key` and `para id` they collate on
- (after declaration) we will check collator's signature and then they can send us any advertisements of collations
- The protocol tracks advertisements received and the source of the advertisement. The advertisement source is the
PeerId of the peer who sent the message. We accept one advertisement per collator per source per relay-parent.
- As a validator, we will handle requests from other subsystems to fetch a collation on a specific ParaId and
relay-parent. These requests are made with the request response protocol CollationFetchingRequest request.
To do so, we need to first check if we have already gathered a collation on that ParaId and relay-parent. If not,
we need to select one of the advertisements and issue a request for it. If we've already issued a request, we shouldn't
issue another one until the first has returned.
- When acting on an advertisement, we issue a Requests::CollationFetchingV1. However, we only request one collation at a time per relay parent. This reduces the bandwidth requirements and as we can second only one candidate per relay parent, the others are probably not required anyway. If the request times out, we need to note the collator as being unreliable and reduce its priority relative to other collators.

*/

/*
Messages over Collation Protocol
enum CollationProtocolV1 {
    CollatorProtocol(CollatorProtocolV1Message),
}
*/

const MaxCollationMessageSize uint64 = 100 * 1024

type CollationProtocolV1 struct {
	// TODO: Implement
	/*
			Messages over Collation Protocol
			enum CollationProtocolV1 {
			    CollatorProtocol(CollatorProtocolV1Message),
			}

		#![allow(unused)]
		fn main() {
		enum CollatorProtocolV1Message {
		    /// Declare the intent to advertise collations under a collator ID and `Para`, attaching a
		    /// signature of the `PeerId` of the node using the given collator ID key.
		    Declare(CollatorId, ParaId, CollatorSignature),
		    /// Advertise a collation to a validator. Can only be sent once the peer has
		    /// declared that they are a collator with given ID.
		    AdvertiseCollation(Hash),
		    /// A collation sent to a validator was seconded.
		    CollationSeconded(SignedFullStatement),
		}
		}
	*/

}

// Type returns CollationMsgType
func (*CollationProtocolV1) Type() byte {
	return network.CollationMsgType
}

// Hash returns the hash of the CollationProtocolV1
func (cp *CollationProtocolV1) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := cp.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (cp *CollationProtocolV1) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*cp)
	if err != nil {
		return enc, err
	}
	return enc, nil
}

func decodeCollationMessage(in []byte) (network.NotificationsMessage, error) {
	collationMessage := CollationProtocolV1{}

	err := scale.Unmarshal(in, &collationMessage)
	if err != nil {
		return nil, fmt.Errorf("cannot decode message: %w", err)
	}

	return &collationMessage, nil
}

func handleCollationMessage(peerID peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a collation message", msg)
	return false, nil
}

func getCollatorHandshake() (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func decodeCollatorHandshake(_ []byte) (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func validateCollatorHandshake(_ peer.ID, _ network.Handshake) error {
	return nil
}

type collatorHandshake struct{}

// String formats a collatorHandshake as a string
func (*collatorHandshake) String() string {
	return "collatorHandshake"
}

// Encode encodes a collatorHandshake message using SCALE
func (*collatorHandshake) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode the message into a collatorHandshake
func (*collatorHandshake) Decode(_ []byte) error {
	return nil
}

// IsValid returns true
func (*collatorHandshake) IsValid() bool {
	return true
}
