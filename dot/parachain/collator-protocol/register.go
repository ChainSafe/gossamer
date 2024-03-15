// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package collatorprotocol

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/parachain/overseer"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func Register(net Network, protocolID protocol.ID, o overseer.OverseerI) (*CollatorProtocolValidatorSide, error) {
	collationFetchingReqResProtocol := net.GetRequestResponseProtocol(
		string(protocolID), collationFetchingRequestTimeout, collationFetchingMaxResponseSize)

	cpvs := CollatorProtocolValidatorSide{
		net:                             net,
		overseer:                        o,
		collationFetchingReqResProtocol: collationFetchingReqResProtocol,
		peerData:                        make(map[peer.ID]PeerData),
	}

	// register collation protocol
	err := net.RegisterNotificationsProtocol(
		protocolID,
		network.CollationMsgType,
		getCollatorHandshake,
		decodeCollatorHandshake,
		validateCollatorHandshake,
		decodeCollationMessage,
		cpvs.handleCollationMessage,
		nil,
		MaxCollationMessageSize,
	)
	if err != nil {
		// try with legacy protocol id
		err1 := net.RegisterNotificationsProtocol(
			protocol.ID(legacyCollationProtocolV1),
			network.CollationMsgType,
			getCollatorHandshake,
			decodeCollatorHandshake,
			validateCollatorHandshake,
			decodeCollationMessage,
			cpvs.handleCollationMessage,
			nil,
			MaxCollationMessageSize,
		)

		if err1 != nil {
			return nil, fmt.Errorf("registering collation protocol, new: %w, legacy:%w", err, err1)
		}
	}

	return &cpvs, nil
}
