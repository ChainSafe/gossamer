package grandpa

type BlockState interface {
	HasHeader(hash common.Hash) (bool, error)
	SubChain(start, end common.Hash) ([]common.Hash, error)
}
