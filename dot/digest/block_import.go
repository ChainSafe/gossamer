package digest

import "github.com/ChainSafe/gossamer/dot/types"

type BlockImportHandler struct {
	epochState   EpochState
	grandpaState GrandpaState
}

func NewBlockImportHandler(epochState EpochState, grandpaState GrandpaState) *BlockImportHandler {
	return &BlockImportHandler{
		epochState:   epochState,
		grandpaState: grandpaState,
	}
}

func (h *BlockImportHandler) Handle(importedBlockHeader *types.Header) error {
	panic("not implemented yet")
}
