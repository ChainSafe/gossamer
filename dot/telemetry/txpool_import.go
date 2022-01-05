// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

// TxpoolImportTM holds `txpool.import` telemetry message, which is supposed to be
// sent when a new transaction gets imported in the transaction pool.
type TxpoolImportTM struct {
	Ready  uint `json:"ready"`
	Future uint `json:"future"`
}

// NewTxpoolImportTM creates a new TxpoolImportTM struct
func NewTxpoolImportTM(ready, future uint) TxpoolImportTM {
	return TxpoolImportTM{
		Ready:  ready,
		Future: future,
	}
}

func (TxpoolImportTM) messageType() string {
	return txPoolImportMsg
}
