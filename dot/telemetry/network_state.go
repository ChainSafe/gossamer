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

import (
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	libp2phost "github.com/libp2p/go-libp2p-core/host"
)

// NetworkStateTM struct to hold network state telemetry messages
type NetworkStateTM struct {
	Msg   string                 `json:"msg"`
	State map[string]interface{} `json:"state"`
}

// NewNetworkStateTM function to create new Network State Telemetry Message
func NewNetworkStateTM(host libp2phost.Host, peerInfos []common.PeerInfo) *NetworkStateTM {
	netState := make(map[string]interface{})
	netState["peerId"] = host.ID()
	hostAddrs := []string{}
	for _, v := range host.Addrs() {
		hostAddrs = append(hostAddrs, v.String())
	}
	netState["externalAddressess"] = hostAddrs
	listAddrs := []string{}
	for _, v := range host.Network().ListenAddresses() {
		listAddrs = append(listAddrs, fmt.Sprintf("%s/p2p/%s", v, host.ID()))
	}
	netState["listenedAddressess"] = listAddrs

	peers := make(map[string]interface{})
	for _, v := range peerInfos {
		p := &peerInfo{
			Roles:      v.Roles,
			BestHash:   v.BestHash.String(),
			BestNumber: v.BestNumber,
		}
		peers[v.PeerID] = *p
	}
	netState["connectedPeers"] = peers

	return &NetworkStateTM{
		Msg:   "system.network_state",
		State: netState,
	}
}

func (tm *NetworkStateTM) messageType() string {
	return tm.Msg
}
