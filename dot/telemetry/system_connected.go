// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import (
	"encoding/json"
	"time"

	"github.com/ChainSafe/gossamer/lib/common"
)

type systemConnectedTM SystemConnected

var _ Message = (*SystemConnected)(nil)

// SystemConnected struct to hold system connected telemetry messages
type SystemConnected struct {
	Authority      bool         `json:"authority"`
	Chain          string       `json:"chain"`
	GenesisHash    *common.Hash `json:"genesis_hash"`
	Implementation string       `json:"implementation"`
	Name           string       `json:"name"`
	NetworkID      string       `json:"network_id"`
	StartupTime    string       `json:"startup_time"`
	Version        string       `json:"version"`
}

// NewSystemConnected function to create new System Connected Telemetry Message
func NewSystemConnected(authority bool, chain string, genesisHash *common.Hash,
	implementation, name, networkID, startupTime, version string) *SystemConnected {
	return &SystemConnected{
		Authority:      authority,
		Chain:          chain,
		GenesisHash:    genesisHash,
		Implementation: implementation,
		Name:           name,
		NetworkID:      networkID,
		StartupTime:    startupTime,
		Version:        version,
	}
}

func (SystemConnected) messageType() string {
	return systemConnectedMsg
}

func (sc SystemConnected) MarshalJSON() ([]byte, error) {
	telemetryData := struct {
		systemConnectedTM
		Timestamp   time.Time `json:"ts"`
		MessageType string    `json:"msg"`
	}{
		Timestamp:         time.Now(),
		MessageType:       sc.messageType(),
		systemConnectedTM: systemConnectedTM(sc),
	}

	return json.Marshal(telemetryData)
}
