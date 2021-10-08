// Copyright 2021 ChainSafe Systems (ON) Corp.
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

package telemetry

import "github.com/ChainSafe/gossamer/lib/common"

// SystemConnectedTM struct to hold system connected telemetry messages
type SystemConnectedTM struct {
	Authority      bool         `json:"authority"`
	Chain          string       `json:"chain"`
	GenesisHash    *common.Hash `json:"genesis_hash"`
	Implementation string       `json:"implementation"`
	Msg            string       `json:"msg"`
	Name           string       `json:"name"`
	NetworkID      string       `json:"network_id"`
	StartupTime    string       `json:"startup_time"`
	Version        string       `json:"version"`
}

// NewSystemConnectedTM function to create new System Connected Telemetry Message
func NewSystemConnectedTM(authority bool, chain string, genesisHash *common.Hash,
	implementation, name, networkID, startupTime, version string) *SystemConnectedTM {
	return &SystemConnectedTM{
		Authority:      authority,
		Chain:          chain,
		GenesisHash:    genesisHash,
		Implementation: implementation,
		Msg:            "system.connected",
		Name:           name,
		NetworkID:      networkID,
		StartupTime:    startupTime,
		Version:        version,
	}
}

func (tm *SystemConnectedTM) messageType() string {
	return tm.Msg
}
