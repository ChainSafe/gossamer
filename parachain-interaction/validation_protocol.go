package parachaininteraction

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxValidationMessageSize uint64 = 100 * 1024

/*
Network Message types her https://paritytech.github.io/polkadot/book/types/network.html#validation-v1

Messages over Validation Protocol
enum ValidationProtocolV1 {
    ApprovalDistribution(ApprovalDistributionV1Message),
    AvailabilityDistribution(AvailabilityDistributionV1Message),
    AvailabilityRecovery(AvailabilityRecoveryV1Message),
    BitfieldDistribution(BitfieldDistributionV1Message),
    PoVDistribution(PoVDistributionV1Message),
    StatementDistribution(StatementDistributionV1Message),
}
*/

func decodeValidationMessage(in []byte) (network.NotificationsMessage, error) {
	// TODO: scale decode the message
	fmt.Println("We got a validation message", in)
	return nil, nil
}

func handleValidationMessage(peerID peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a validation message", msg)
	return false, nil
}
