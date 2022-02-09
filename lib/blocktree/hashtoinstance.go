package blocktree

import (
	"sync"

	"github.com/ChainSafe/gossamer/lib/runtime"
)

type hashToInstance struct {
	mutex   sync.RWMutex
	mapping map[Hash]runtime.Instance
}

func newHashToInstance() *hashToInstance {
	return &hashToInstance{
		mapping: make(map[Hash]runtime.Instance),
	}
}

func (h *hashToInstance) get(hash Hash) (instance runtime.Instance) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.mapping[hash]
}

func (h *hashToInstance) set(hash Hash, instance runtime.Instance) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.mapping[hash] = instance
}

func (h *hashToInstance) delete(hash Hash) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(h.mapping, hash)
}
