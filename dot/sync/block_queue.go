package sync

import (
	"errors"
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var errQueueFull = errors.New("cannot push item; queue is at capacity")

type blockQueue struct {
	sync.RWMutex
	cap int
	ch chan *types.BlockData
	blocks map[common.Hash]*types.BlockData
}

// newBlockQueue initializes a queue of *types.BlockData with the given capacity.
func newBlockQueue(cap int) *blockQueue {
	return &blockQueue{
		cap: cap,
		ch: make(chan *types.BlockData, cap),
		blocks: make(map[common.Hash]*types.BlockData),
	}
}

func (q *blockQueue) push(bd *types.BlockData) {
	q.Lock()
	defer q.Unlock()

	q.blocks[bd.Hash] = bd
	q.ch <- bd
}

func (q *blockQueue) pop() *types.BlockData {
	q.Lock()
	defer q.Unlock()

 	bd := <-q.ch
	delete(q.blocks, bd.Hash)
	return bd	
}

func (q *blockQueue) has(hash common.Hash) bool {
	q.RLock()
	defer q.RUnlock()
	_, has := q.blocks[hash]
	return has
}

func (q *blockQueue) get(hash common.Hash) *types.BlockData {
	q.RLock()
	defer q.RUnlock()	
	return q.blocks[hash]
}