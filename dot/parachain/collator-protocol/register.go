package collatorprotocol

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func Register(net Network, protocolID protocol.ID, overseerChan chan<- any) (*CollatorProtocolValidatorSide, error) {

	// TODO: fill up values for CollatorProtocolValidatorSide
	cpvs := CollatorProtocolValidatorSide{
		SubSystemToOverseer: overseerChan,
	}

	// register collation protocol
	var err error
	err = net.RegisterNotificationsProtocol(
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
			protocol.ID(LEGACY_COLLATION_PROTOCOL_V1),
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
			err = fmt.Errorf("registering collation protocol, new: %w, legacy:%w", err, err1)
		}
	}

	if err != nil {
		return nil, err
	}

	return &cpvs, nil
}
