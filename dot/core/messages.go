// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package core

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/transaction"

	"github.com/libp2p/go-libp2p/core/peer"
)

func (s *Service) validateTransaction(head *types.Header, rt runtime.Instance,
	tx types.Extrinsic) (validity *transaction.Validity, err error) {
	s.storageState.Lock()

	ts, err := s.storageState.TrieState(&head.StateRoot)
	s.storageState.Unlock()
	if err != nil {
		return nil, fmt.Errorf("cannot get trie state from storage for root %s: %w", head.StateRoot, err)
	}

	rt.SetContextStorage(ts)

	// validate each transaction
	externalExt, err := s.buildExternalTransaction(rt, tx)
	if err != nil {
		return nil, fmt.Errorf("building external transaction: %w", err)
	}

	validity, err = rt.ValidateTransaction(externalExt)
	if err != nil {
		logger.Debugf("failed to validate transaction: %s", err)
		return nil, err
	}

	vtx := transaction.NewValidTransaction(tx, validity)

	// push to the transaction queue of BABE session
	hash := s.transactionState.AddToPool(vtx)
	logger.Tracef("added transaction with hash %s to pool", hash)

	return validity, nil
}

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

	bestBlockHash := head.Hash()
	rt, err := s.blockState.GetRuntime(bestBlockHash)
	if err != nil {
		return false, err
	}

	allTxnsAreValid := true
	for _, tx := range txs {
		validity, err := s.validateTransaction(head, rt, tx)
		if err != nil {
			allTxnsAreValid = false
			switch err.(type) {
			case runtime.InvalidTransaction:
				s.net.ReportPeer(peerset.ReputationChange{
					Value:  peerset.BadTransactionValue,
					Reason: peerset.BadTransactionReason,
				}, peerID)
			case runtime.UnknownTransaction:
			default:
				return false, fmt.Errorf("validating transaction from peerID %s: %w", peerID, err)
			}
			continue
		}

		if validity.Propagate {
			// find tx(s) that should propagate
			toPropagate = append(toPropagate, tx)
		}
	}

	if allTxnsAreValid {
		s.net.ReportPeer(peerset.ReputationChange{
			Value:  peerset.GoodTransactionValue,
			Reason: peerset.GoodTransactionReason,
		}, peerID)
	}

	msg.Extrinsics = toPropagate
	return len(msg.Extrinsics) > 0, nil
}

// TransactionsCount returns number for pending transactions in pool
func (s *Service) TransactionsCount() int {
	return len(s.transactionState.PendingInPool())
}
