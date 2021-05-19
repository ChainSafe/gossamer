// Copyright 2020 ChainSafe Systems (ON) Corp.
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
package subscription

import (
	"fmt"
	"reflect"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// Listener interface for functions that define Listener related functions
type Listener interface {
	Listen()
}

// WSConnAPI interface defining methors a WSConn should have
type WSConnAPI interface {
	safeSend(interface{})
}

// StorageObserver struct to hold data for observer (Observer Design Pattern)
type StorageObserver struct {
	id     uint
	filter map[string][]byte
	wsconn WSConnAPI
}

// Change type defining key value pair representing change
type Change [2]string

// ChangeResult struct to hold change result data
type ChangeResult struct {
	Changes []Change `json:"changes"`
	Block   string   `json:"block"`
}

// Update is called to notify observer of new value
func (s *StorageObserver) Update(change *state.SubscriptionResult) {
	if change == nil {
		return
	}

	changeResult := ChangeResult{
		Block:   change.Hash.String(),
		Changes: make([]Change, len(change.Changes)),
	}
	for i, v := range change.Changes {
		changeResult.Changes[i] = Change{common.BytesToHex(v.Key), common.BytesToHex(v.Value)}
	}

	res := newSubcriptionBaseResponseJSON()
	res.Method = "state_storage"
	res.Params.Result = changeResult
	res.Params.SubscriptionID = s.GetID()
	s.wsconn.safeSend(res)
}

// GetID the id for the Observer
func (s *StorageObserver) GetID() uint {
	return s.id
}

// GetFilter returns the filter the Observer is using
func (s *StorageObserver) GetFilter() map[string][]byte {
	return s.filter
}

// Listen to satisfy Listener interface (but is no longer used by StorageObserver)
func (s *StorageObserver) Listen() {}

// BlockListener to handle listening for blocks importedChan
type BlockListener struct {
	Channel chan *types.Block
	wsconn  WSConnAPI
	ChanID  byte
	subID   uint
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockListener) Listen() {
	for block := range l.Channel {
		if block == nil {
			continue
		}
		head, err := modules.HeaderToJSON(*block.Header)
		if err != nil {
			logger.Error("failed to convert header to JSON", "error", err)
		}

		res := newSubcriptionBaseResponseJSON()
		res.Method = "chain_newHead"
		res.Params.Result = head
		res.Params.SubscriptionID = l.subID
		l.wsconn.safeSend(res)
	}
}

// BlockFinalizedListener to handle listening for finalised blocks
type BlockFinalizedListener struct {
	channel chan *types.FinalisationInfo
	wsconn  WSConnAPI
	chanID  byte
	subID   uint
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockFinalizedListener) Listen() {
	for info := range l.channel {
		if info == nil || info.Header == nil {
			continue
		}
		head, err := modules.HeaderToJSON(*info.Header)
		if err != nil {
			logger.Error("failed to convert header to JSON", "error", err)
		}
		res := newSubcriptionBaseResponseJSON()
		res.Method = "chain_finalizedHead"
		res.Params.Result = head
		res.Params.SubscriptionID = l.subID
		l.wsconn.safeSend(res)
	}
}

// ExtrinsicSubmitListener to handle listening for extrinsic events
type ExtrinsicSubmitListener struct {
	wsconn    WSConnAPI
	subID     uint
	extrinsic types.Extrinsic

	importedChan    chan *types.Block
	importedChanID  byte
	importedHash    common.Hash
	finalisedChan   chan *types.FinalisationInfo
	finalisedChanID byte
}

// AuthorExtrinsicUpdates method name
const AuthorExtrinsicUpdates = "author_extrinsicUpdate"

// Listen implementation of Listen interface to listen for importedChan changes
func (l *ExtrinsicSubmitListener) Listen() {
	// listen for imported blocks with extrinsic
	go func() {
		for block := range l.importedChan {
			if block == nil {
				continue
			}
			bodyHasExtrinsic, err := block.Body.HasExtrinsic(l.extrinsic)
			if err != nil {
				fmt.Printf("error %v\n", err)
			}

			if bodyHasExtrinsic {
				resM := make(map[string]interface{})
				resM["inBlock"] = block.Header.Hash().String()

				l.importedHash = block.Header.Hash()
				l.wsconn.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, l.subID, resM))
			}
		}
	}()

	// listen for finalised headers
	go func() {
		for info := range l.finalisedChan {
			if reflect.DeepEqual(l.importedHash, info.Header.Hash()) {
				resM := make(map[string]interface{})
				resM["finalised"] = info.Header.Hash().String()
				l.wsconn.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, l.subID, resM))
			}
		}
	}()
}

// RuntimeVersionListener to handle listening for Runtime Version
type RuntimeVersionListener struct {
	wsconn *WSConn
	subID  uint
}

// Listen implementation of Listen interface to listen for runtime version changes
func (l *RuntimeVersionListener) Listen() {
	// This sends current runtime version once when subscription is created
	// TODO (ed) add logic to send updates when runtime version changes
	rtVersion, err := l.wsconn.CoreAPI.GetRuntimeVersion(nil)
	if err != nil {
		return
	}
	ver := modules.StateRuntimeVersionResponse{}

	ver.SpecName = string(rtVersion.SpecName())
	ver.ImplName = string(rtVersion.ImplName())
	ver.AuthoringVersion = rtVersion.AuthoringVersion()
	ver.SpecVersion = rtVersion.SpecVersion()
	ver.ImplVersion = rtVersion.ImplVersion()
	ver.TransactionVersion = rtVersion.TransactionVersion()
	ver.Apis = modules.ConvertAPIs(rtVersion.APIItems())

	l.wsconn.safeSend(newSubscriptionResponse("state_runtimeVersion", l.subID, ver))
}
