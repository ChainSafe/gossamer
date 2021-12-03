// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

// txpoolImportTM holds `txpool.import` telemetry message, which is supposed to be
// sent when a new transaction gets imported in the transaction pool.
type txpoolImportTM struct {
	Ready  uint `json:"ready"`
	Future uint `json:"future"`
}

// NewTxpoolImportTM creates a new txpoolImportTM struct
func NewTxpoolImportTM(ready, future uint) Message {
	return &txpoolImportTM{
		Ready:  ready,
		Future: future,
	}
}

func (txpoolImportTM) messageType() string {
	return txPoolImportMsg
}
