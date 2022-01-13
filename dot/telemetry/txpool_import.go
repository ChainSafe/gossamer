// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"
)

type txpoolImportTM TxpoolImport

var _ Message = (*TxpoolImport)(nil)

// TxpoolImport holds `txpool.import` telemetry message, which is supposed to be
// sent when a new transaction gets imported in the transaction pool.
type TxpoolImport struct {
	Ready  uint `json:"ready"`
	Future uint `json:"future"`
}

// NewTxpoolImport creates a new TxpoolImportTM struct
func NewTxpoolImport(ready, future uint) *TxpoolImport {
	return &TxpoolImport{
		Ready:  ready,
		Future: future,
	}
}

func (TxpoolImport) Type() string {
	return txPoolImportMsg
}

func (tx TxpoolImport) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		txpoolImportTM
		MessageType string    `json:"msg"`
		Timestamp   time.Time `json:"ts"`
	}{
		Timestamp:      time.Now(),
		MessageType:    tx.Type(),
		txpoolImportTM: txpoolImportTM(tx),
	}

	return json.Marshal(telemetryData)
}
