package node

import "github.com/ChainSafe/gossamer/lib/common"

type Putter interface {
	Put(key, value []byte) (err error)
}

type Getter interface {
	Get(key []byte) (value []byte, err error)
}

type DeltaSubValueRecorder interface {
	RecordDeleted(subValueHash common.Hash)
}
