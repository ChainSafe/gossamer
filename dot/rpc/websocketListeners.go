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
package rpc

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

func (c *WSConn) startListener(lid int) {
	go c.subscriptions[lid].Listen()
}

func (c *WSConn) initStorageChangeListener(reqID float64, params interface{}) (int, error) {
	if c.storageAPI == nil {
		c.safeSendError(reqID, nil, "error StorageAPI not set")
		return 0, fmt.Errorf("error StorageAPI not set")
	}

	scl := &StorageChangeListener{
		channel: make(chan *state.SubscriptionResult),
		wsconn:  c,
	}
	sub := &state.StorageSubscription{
		Filter:   make(map[string][]byte),
		Listener: scl.channel,
	}

	pA, ok := params.([]interface{})
	if !ok {
		return 0, fmt.Errorf("unknow parameter type")
	}
	for _, param := range pA {
		switch p := param.(type) {
		case []interface{}:
			for _, pp := range param.([]interface{}) {
				data, ok := pp.(string)
				if !ok {
					return 0, fmt.Errorf("unknow parameter type")
				}
				sub.Filter[data] = []byte{}
			}
		case string:
			sub.Filter[p] = []byte{}
		default:
			return 0, fmt.Errorf("unknow parameter type")
		}
	}

	chanID, err := c.storageAPI.RegisterStorageChangeChannel(*sub)
	if err != nil {
		return 0, err
	}
	scl.chanID = chanID

	c.qtyListeners++
	scl.subID = c.qtyListeners
	c.subscriptions[scl.subID] = scl
	c.storageSubChannels[scl.subID] = chanID

	initRes := newSubscriptionResponseJSON(scl.subID, reqID)
	c.safeSend(initRes)

	return scl.subID, nil
}

// StorageChangeListener for listening to state change channels
type StorageChangeListener struct {
	channel chan *state.SubscriptionResult
	wsconn  WSConnAPI
	chanID  byte
	subID   int
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *StorageChangeListener) Listen() {
	for change := range l.channel {
		if change == nil {
			continue
		}

		result := make(map[string]interface{})
		result["block"] = change.Hash.String()
		changes := make([][]string, 0, len(change.Changes))
		for _, v := range change.Changes {
			kv := []string{common.BytesToHex(v.Key), common.BytesToHex(v.Value)}
			changes = append(changes, kv)
		}
		result["changes"] = changes

		res := newSubcriptionBaseResponseJSON()
		res.Method = "state_storage"
		res.Params.Result = result
		res.Params.SubscriptionID = l.subID
		l.wsconn.safeSend(res)
	}
}

// BlockListener to handle listening for blocks importedChan
type BlockListener struct {
	channel chan *types.Block
	wsconn  WSConnAPI
	chanID  byte
	subID   int
}

func (c *WSConn) initBlockListener(reqID float64) (int, error) {
	bl := &BlockListener{
		channel: make(chan *types.Block),
		wsconn:  c,
	}

	if c.blockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	chanID, err := c.blockAPI.RegisterImportedChannel(bl.channel)
	if err != nil {
		return 0, err
	}
	bl.chanID = chanID
	c.qtyListeners++
	bl.subID = c.qtyListeners
	c.subscriptions[bl.subID] = bl
	c.blockSubChannels[bl.subID] = chanID
	initRes := newSubscriptionResponseJSON(bl.subID, reqID)
	c.safeSend(initRes)

	return bl.subID, nil
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockListener) Listen() {
	for block := range l.channel {
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

// BlockFinalizedListener to handle listening for finalized blocks
type BlockFinalizedListener struct {
	channel chan *types.Header
	wsconn  WSConnAPI
	chanID  byte
	subID   int
}

func (c *WSConn) initBlockFinalizedListener(reqID float64) (int, error) {
	bfl := &BlockFinalizedListener{
		channel: make(chan *types.Header),
		wsconn:  c,
	}

	if c.blockAPI == nil {
		c.safeSendError(reqID, nil, "error BlockAPI not set")
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	chanID, err := c.blockAPI.RegisterFinalizedChannel(bfl.channel)
	if err != nil {
		return 0, err
	}
	bfl.chanID = chanID
	c.qtyListeners++
	bfl.subID = c.qtyListeners
	c.subscriptions[bfl.subID] = bfl
	c.blockSubChannels[bfl.subID] = chanID
	initRes := newSubscriptionResponseJSON(bfl.subID, reqID)
	c.safeSend(initRes)

	return bfl.subID, nil
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockFinalizedListener) Listen() {
	for header := range l.channel {
		if header == nil {
			continue
		}
		head, err := modules.HeaderToJSON(*header)
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
	subID     int
	extrinsic types.Extrinsic

	importedChan    chan *types.Block
	importedChanID  byte
	importedHash    common.Hash
	finalizedChan   chan *types.Header
	finalizedChanID byte
}

// AuthorExtrinsicUpdates method name
const AuthorExtrinsicUpdates = "author_extrinsicUpdate"

func (c *WSConn) initExtrinsicWatch(reqID float64, params interface{}) (int, error) {
	pA := params.([]interface{})
	extBytes, err := common.HexToBytes(pA[0].(string))
	if err != nil {
		return 0, err
	}

	// listen for built blocks
	esl := &ExtrinsicSubmitListener{
		importedChan:  make(chan *types.Block),
		wsconn:        c,
		extrinsic:     types.Extrinsic(extBytes),
		finalizedChan: make(chan *types.Header),
	}

	if c.blockAPI == nil {
		return 0, fmt.Errorf("error BlockAPI not set")
	}
	esl.importedChanID, err = c.blockAPI.RegisterImportedChannel(esl.importedChan)
	if err != nil {
		return 0, err
	}

	esl.finalizedChanID, err = c.blockAPI.RegisterFinalizedChannel(esl.finalizedChan)
	if err != nil {
		return 0, err
	}

	c.qtyListeners++
	esl.subID = c.qtyListeners
	c.subscriptions[esl.subID] = esl
	c.blockSubChannels[esl.subID] = esl.importedChanID

	err = c.coreAPI.HandleSubmittedExtrinsic(extBytes)
	if err != nil {
		return 0, err
	}
	c.safeSend(newSubscriptionResponseJSON(esl.subID, reqID))

	// TODO (ed) since HandleSubmittedExtrinsic has been called we assume the extrinsic is in the tx queue
	//  should we add a channel to tx queue so we're notified when it's in the queue
	if c.coreAPI.IsBlockProducer() {
		c.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, esl.subID, "ready"))
	}

	// todo (ed) determine which peer extrinsic has been broadcast to, and set status
	return esl.subID, err
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *ExtrinsicSubmitListener) Listen() {
	// listen for imported blocks with extrinsic
	go func() {
		for block := range l.importedChan {
			if block == nil {
				continue
			}
			exts, err := block.Body.AsExtrinsics()
			if err != nil {
				fmt.Printf("error %v\n", err)
			}
			for _, v := range exts {
				if reflect.DeepEqual(v, l.extrinsic) {
					resM := make(map[string]interface{})
					resM["inBlock"] = block.Header.Hash().String()

					l.importedHash = block.Header.Hash()
					l.wsconn.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, l.subID, resM))
				}
			}
		}
	}()

	// listen for finalized headers
	go func() {
		for header := range l.finalizedChan {
			if reflect.DeepEqual(l.importedHash, header.Hash()) {
				resM := make(map[string]interface{})
				resM["finalized"] = header.Hash().String()
				l.wsconn.safeSend(newSubscriptionResponse(AuthorExtrinsicUpdates, l.subID, resM))
			}
		}
	}()
}

// RuntimeVersionListener to handle listening for Runtime Version
type RuntimeVersionListener struct {
	wsconn *WSConn
	subID  int
}

func (c *WSConn) initRuntimeVersionListener(reqID float64) (int, error) {
	rvl := &RuntimeVersionListener{
		wsconn: c,
	}
	if c.coreAPI == nil {
		c.safeSendError(reqID, nil, "error CoreAPI not set")
		return 0, fmt.Errorf("error CoreAPI not set")
	}
	c.qtyListeners++
	rvl.subID = c.qtyListeners
	c.subscriptions[rvl.subID] = rvl
	initRes := newSubscriptionResponseJSON(rvl.subID, reqID)
	c.safeSend(initRes)

	return rvl.subID, nil
}

// Listen implementation of Listen interface to listen for runtime version changes
func (l *RuntimeVersionListener) Listen() {
	// This sends current runtime version once when subscription is created
	// TODO (ed) add logic to send updates when runtime version changes
	rtVersion, err := l.wsconn.coreAPI.GetRuntimeVersion(nil)
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
