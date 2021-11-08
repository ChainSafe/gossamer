// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"errors"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

// HandleTransactionMessage validates each transaction in the message and
// adds valid transactions to the transaction queue of the BABE session
// returns boolean for transaction propagation, true - transactions should be propagated
func (s *Service) HandleTransactionMessage(peerID peer.ID, msg *network.TransactionMessage) (bool, error) {
	logger.Debug("received TransactionMessage")

	if !s.net.IsSynced() {
		logger.Debug("ignoring TransactionMessage, not yet synced")
		return false, nil
	}

	// get transactions from message extrinsics
	txs := msg.Extrinsics
	var toPropagate []types.Extrinsic

	head, err := s.blockState.BestBlockHeader()
	if err != nil {
		return false, err
	}

	hash := head.Hash()
	rt, err := s.blockState.GetRuntime(&hash)
	if err != nil {
		return false, err
	}

	for _, tx := range txs {
		err = func() error {
			s.storageState.Lock()
			defer s.storageState.Unlock()

			ts, err := s.storageState.TrieState(&head.StateRoot) //nolint
			if err != nil {
				return err
			}

			rt.SetContextStorage(ts)

			// validate each transaction
			externalExt := types.Extrinsic(append([]byte{byte(types.TxnExternal)}, tx...))
			val, err := rt.ValidateTransaction(externalExt)
			if err != nil {
				if errors.Is(err, runtime.ErrInvalidTransaction) {
					s.net.ReportPeer(peerset.ReputationChange{
						Value:  peerset.BadTransactionValue,
						Reason: peerset.BadTransactionReason,
					}, peerID)
				}
				logger.Debugf("failed to validate transaction: %s", err)
				return nil
			}

			// create new valid transaction
			vtx := transaction.NewValidTransaction(tx, val)

			// push to the transaction queue of BABE session
			hash := s.transactionState.AddToPool(vtx)
			logger.Tracef("added transaction with hash %s to pool", hash)

			// find tx(s) that should propagate
			if val.Propagate {
				toPropagate = append(toPropagate, tx)
			}

			return nil
		}()

		if err != nil {
			return false, err
		}
	}

	s.net.ReportPeer(peerset.ReputationChange{
		Value:  peerset.GoodTransactionValue,
		Reason: peerset.GoodTransactionReason,
	}, peerID)

	msg.Extrinsics = toPropagate
	return len(msg.Extrinsics) > 0, nil
}

// TransactionsCount returns number for pending transactions in pool
func (s *Service) TransactionsCount() int {
	return len(s.transactionState.PendingInPool())
}
