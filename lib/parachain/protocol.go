package parachain

import (
	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
)

type View struct {
	Heads           []common.Hash
	FinalizedNumber parachaintypes.BlockNumber
}
