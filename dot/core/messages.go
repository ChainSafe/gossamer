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

package core

import (
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// HandleTransactionMessage validates each transaction in the message and
// adds valid transactions to the transaction queue of the BABE session
// returns boolean for transaction propagation, true - transactions should be propagated
func (s *Service) HandleTransactionMessage(msg *network.TransactionMessage) (bool, error) {
	logger.Debug("received TransactionMessage")

	// get transactions from message extrinsics
	txs := msg.Extrinsics
	var toPropagate []types.Extrinsic

	rt, err := s.blockState.GetRuntime(nil)
	if err != nil {
		return false, err
	}

	for _, tx := range txs {
		ts, err := s.storageState.TrieState(nil)
		if err != nil {
			return false, err
		}

		rt.SetContextStorage(ts)

		// validate each transaction
		externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, tx...))
		val, err := rt.ValidateTransaction(externalExt)
		if err != nil {
			logger.Debug("failed to validate transaction", "err", err)
			continue
		}

		// create new valid transaction
		vtx := transaction.NewValidTransaction(tx, val)

		// push to the transaction queue of BABE session
		hash := s.transactionState.AddToPool(vtx)
		logger.Trace("Added transaction to queue", "hash", hash)

		// find tx(s) that should propagate
		if val.Propagate {
			toPropagate = append(toPropagate, tx)
		}
	}

	msg.Extrinsics = toPropagate

	return len(msg.Extrinsics) > 0, nil
}

// TransactionsCount returns number for pending transactions in pool
func (s *Service) TransactionsCount() int {
	return len(s.transactionState.PendingInPool())
}
