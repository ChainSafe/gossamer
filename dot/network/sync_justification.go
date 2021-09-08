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

// import (
// 	"math/big"

// 	"github.com/libp2p/go-libp2p-core/peer"
// )

// // SendJustificationRequest pushes a justification request to the queue to be sent out to the network
// func (s *Service) SendJustificationRequest(to peer.ID, num uint32) {
// 	s.syncQueue.pushJustificationRequest(to, uint64(num))
// }

// func (q *syncQueue) pushJustificationRequest(to peer.ID, start uint64) {
// 	startHash, err := q.s.blockState.GetHashByNumber(big.NewInt(int64(start)))
// 	if err != nil {
// 		logger.Debug("failed to get hash for block w/ number", "number", start, "error", err)
// 		return
// 	}

// 	req := createBlockRequestWithHash(startHash, blockRequestSize)
// 	req.RequestedData = RequestedDataJustification

// 	logger.Debug("pushing justification request to queue", "start", start, "hash", startHash)
// 	q.justificationRequestData.Store(startHash, requestData{
// 		received: false,
// 	})

// 	q.requestCh <- &syncRequest{
// 		req: req,
// 		to:  to,
// 	}
// }
