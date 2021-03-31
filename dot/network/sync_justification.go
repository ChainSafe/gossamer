package network

import (
	"math/big"
	"time"
)

func (q *syncQueue) finalizeAtHead() {
	prev, err := q.s.blockState.GetFinalizedHeader(0, 0)
	if err != nil {
		logger.Error("failed to get latest finalized block header", "error", err)
		return
	}

	for {
		select {
		// sleep for average block time TODO: make this configurable from slot duration
		case <-time.After(time.Second * 12):
		case <-q.ctx.Done():
			return
		}

		curr, err := q.s.blockState.GetFinalizedHeader(0, 0)
		if err != nil {
			continue
		}

		logger.Debug("checking finalized blocks", "curr", curr.Number, "prev", prev.Number)

		if curr.Number.Cmp(prev.Number) > 0 {
			prev = curr
			continue
		}

		// no new blocks have been finalized, request block justifications from peers
		head, err := q.s.blockState.BestBlockNumber()
		if err != nil {
			prev = curr
			continue
		}

		prev = curr

		start := head.Uint64() - uint64(blockRequestSize)
		if curr.Number.Uint64() > start {
			start = curr.Number.Uint64() + 1
		}

		q.pushJustificationRequest(start)
	}
}

func (q *syncQueue) pushJustificationRequest(start uint64) {
	startHash, err := q.s.blockState.GetHashByNumber(big.NewInt(int64(start)))
	if err != nil {
		logger.Error("failed to get hash for block w/ number", "number", start, "error", err)
		return
	}

	req := createBlockRequestWithHash(startHash, blockRequestSize)
	req.RequestedData = RequestedDataJustification

	logger.Debug("pushing justification request to queue", "start", start)
	q.justificationRequestData.Store(startHash, requestData{
		received: false,
	})

	q.requestCh <- &syncRequest{
		req: req,
		to:  "",
	}
}
