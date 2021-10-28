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

// networkStateTM struct to hold network state telemetry messages
type networkStateTM struct {
	State map[string]interface{} `json:"state"`
}

// NewNetworkStateTM function to create new Network State Telemetry Message
func NewNetworkStateTM(host libp2phost.Host, peerInfos []common.PeerInfo) Message {
	netState := make(map[string]interface{})
	netState["peerId"] = host.ID()
	hostAddrs := make([]string, 0, len(host.Addrs()))
	for _, v := range host.Addrs() {
		hostAddrs = append(hostAddrs, v.String())
	}
	netState["externalAddressess"] = hostAddrs

	netListAddrs := host.Network().ListenAddresses()
	listAddrs := make([]string, 0, len(netListAddrs))
	for _, v := range netListAddrs {
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

	return &networkStateTM{
		State: netState,
	}
}

func (networkStateTM) messageType() string {
	return systemNetworkStateMsg
}
