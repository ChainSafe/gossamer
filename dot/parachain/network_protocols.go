// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	LEGACY_VALIDATION_PROTOCOL_V1 = "/polkadot/validation/1"
)

type ReqProtocolName uint

const (
	ChunkFetchingV1 ReqProtocolName = iota
	CollationFetchingV1
	PoVFetchingV1
	AvailableDataFetchingV1
	StatementFetchingV1
	DisputeSendingV1
)

type PeerSetProtocolName uint

const (
	ValidationProtocolName PeerSetProtocolName = iota
	CollationProtocolName
)

func GenerateReqProtocolName(protocol ReqProtocolName, forkID string, GenesisHash common.Hash) string {
	prefix := fmt.Sprintf("/%s", GenesisHash.String())

	if forkID != "" {
		prefix = fmt.Sprintf("%s/%s", prefix, forkID)
	}

	switch protocol {
	case ChunkFetchingV1:
		return fmt.Sprintf("%s/req_chunk/1", prefix)
	case CollationFetchingV1:
		return fmt.Sprintf("%s/req_collation/1", prefix)
	case PoVFetchingV1:
		return fmt.Sprintf("%s/req_pov/1", prefix)
	case AvailableDataFetchingV1:
		return fmt.Sprintf("%s/req_available_data/1", prefix)
	case StatementFetchingV1:
		return fmt.Sprintf("%s/req_statement/1", prefix)
	case DisputeSendingV1:
		return fmt.Sprintf("%s/send_dispute/1", prefix)
	default:
		panic("unknown protocol")
	}
}

func GeneratePeersetProtocolName(protocol PeerSetProtocolName, forkID string, GenesisHash common.Hash, version uint32,
) string {
	genesisHash := GenesisHash.String()
	genesisHash = strings.TrimPrefix(genesisHash, "0x")

	prefix := fmt.Sprintf("/%s", genesisHash)

	if forkID != "" {
		prefix = fmt.Sprintf("%s/%s", prefix, forkID)
	}

	switch protocol {
	case ValidationProtocolName:
		return fmt.Sprintf("%s/validation/%d", prefix, version)
		// message over this protocol is BitfieldDistributionMessage, StatementDistributionMessage,
		// ApprovalDistributionMessage
	case CollationProtocolName:
		return fmt.Sprintf("%s/collation/%d", prefix, version)
		// message over this protocol is CollatorProtocolMessage
	default:
		panic("unknown protocol")
	}
}
