package overseer

import "github.com/ChainSafe/gossamer/lib/common"

type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
}

type ActiveLeavesUpdate struct {
	Activated *ActivatedLeaf
}
