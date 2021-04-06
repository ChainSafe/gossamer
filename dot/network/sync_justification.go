// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

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
		case <-time.After(q.slotDuration * 2):
		case <-q.ctx.Done():
			return
		}

		head, err := q.s.blockState.BestBlockNumber()
		if err != nil {
			continue
		}

		if head.Int64() < q.goal {
			continue
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

		prev = curr

		start := head.Uint64() - uint64(blockRequestSize)
		if curr.Number.Uint64() > start {
			start = curr.Number.Uint64() + 1
		} else if int(start) < int(blockRequestSize) {
			start = 1
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

	logger.Debug("pushing justification request to queue", "start", start, "hash", startHash)
	q.justificationRequestData.Store(startHash, requestData{
		received: false,
	})

	q.requestCh <- &syncRequest{
		req: req,
		to:  "",
	}
}
