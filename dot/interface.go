package dot

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/services"
)

// BlockProducer is the interface that a block production service must implement
type BlockProducer interface {
	services.Service

	BlockProduced() <-chan types.Block
	SetBlockProduced(<-chan types.Block)
	SetLock(*sync.Mutex) // TODO: can Pause be used instead?
	Pause()
	SetRuntime(*runtime.Runtime)
}
