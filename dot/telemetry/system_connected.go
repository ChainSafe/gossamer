// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package telemetry

import "github.com/ChainSafe/gossamer/lib/common"

// SystemConnectedTM struct to hold system connected telemetry messages
type SystemConnectedTM struct {
	Authority      bool         `json:"authority"`
	Chain          string       `json:"chain"`
	GenesisHash    *common.Hash `json:"genesis_hash"`
	Implementation string       `json:"implementation"`
	Name           string       `json:"name"`
	NetworkID      string       `json:"network_id"`
	StartupTime    string       `json:"startup_time"`
	Version        string       `json:"version"`
}

// NewSystemConnectedTM function to create new System Connected Telemetry Message
func NewSystemConnectedTM(authority bool, chain string, genesisHash *common.Hash,
	implementation, name, networkID, startupTime, version string) SystemConnectedTM {
	return SystemConnectedTM{
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

func (SystemConnectedTM) messageType() string {
	return systemConnectedMsg
}
