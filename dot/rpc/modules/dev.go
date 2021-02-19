// Copyright 2019 ChainSafe Systems (ON) Corp.
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

package modules

import (
	"encoding/binary"
	"errors"
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

var blockProducerStoppedMsg = "babe service stopped"
var blockProducerStartedMsg = "babe service started"
var networkStoppedMsg = "network service stopped"
var networkStartedMsg = "network service started"

// DevModule is an RPC module that provides developer endpoints
type DevModule struct {
	networkAPI       NetworkAPI
	blockProducerAPI BlockProducerAPI
}

// NewDevModule creates a new Dev module.
func NewDevModule(bp BlockProducerAPI, net NetworkAPI) *DevModule {
	return &DevModule{
		networkAPI:       net,
		blockProducerAPI: bp,
	}
}

// Control to send start and stop messages to services
func (m *DevModule) Control(r *http.Request, req *[]string, res *string) error {
	reqA := *req
	var err error
	switch reqA[0] {
	case "babe":
		if m.blockProducerAPI == nil {
			return errors.New("not a block producer")
		}

		switch reqA[1] {
		case "stop":
			err = m.blockProducerAPI.Pause()
			*res = blockProducerStoppedMsg
		case "start":
			err = m.blockProducerAPI.Resume()
			*res = blockProducerStartedMsg
		}

	case "network":
		switch reqA[1] {
		case "stop":
			err = m.networkAPI.Stop()
			*res = networkStoppedMsg
		case "start":
			err = m.networkAPI.Start()
			*res = networkStartedMsg
		}
	}
	return err
}

// SlotDuration Dev RPC to return slot duration
func (m *DevModule) SlotDuration(r *http.Request, req *EmptyRequest, res *string) error {
	var err error
	*res = uint64ToHex(m.blockProducerAPI.SlotDuration())
	return err
}

// EpochLength Dev RPC to return epoch length
func (m *DevModule) EpochLength(r *http.Request, req *EmptyRequest, res *string) error {
	var err error
	*res = uint64ToHex(m.blockProducerAPI.EpochLength())
	return err
}

// uint64ToHex converts a uint64 to a hexed string
func uint64ToHex(input uint64) string {
	buffer := make([]byte, 8)
	binary.LittleEndian.PutUint64(buffer, input)
	return common.BytesToHex(buffer)
}
