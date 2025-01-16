// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"fmt"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
)

type PeerSetProtocolName uint

const (
	ValidationProtocolName PeerSetProtocolName = iota
	CollationProtocolName
)

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
