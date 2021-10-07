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
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

const (
	grandpaJustificationsMethod  = "grandpa_justifications"
	stateRuntimeVersionMethod    = "state_runtimeVersion"
	authorExtrinsicUpdatesMethod = "author_extrinsicUpdate"
	chainFinalizedHeadMethod     = "chain_finalizedHead"
	chainNewHeadMethod           = "chain_newHead"
	chainAllHeadMethod           = "chain_allHead"
	stateStorageMethod           = "state_storage"
)

var (
	// ErrCannotCancel when is not possible to cancel a goroutine after `cancelTimeout` seconds
	ErrCannotCancel = errors.New("cannot cancel listening goroutines")

	defaultCancelTimeout = time.Second * 10
)

// Listener interface for functions that define Listener related functions
type Listener interface {
	Listen()
	Stop() error
}

// WSConnAPI interface defining methors a WSConn should have
type WSConnAPI interface {
	safeSend(interface{})
}

// StorageObserver struct to hold data for observer (Observer Design Pattern)
type StorageObserver struct {
	id     uint32
	filter map[string][]byte
	wsconn *WSConn
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
	res.Method = stateStorageMethod
	res.Params.Result = changeResult
	res.Params.SubscriptionID = s.id
	s.wsconn.safeSend(res)
}

// GetID the id for the Observer
func (s *StorageObserver) GetID() uint {
	return uint(s.id)
}

// GetFilter returns the filter the Observer is using
func (s *StorageObserver) GetFilter() map[string][]byte {
	return s.filter
}

// Listen to satisfy Listener interface (but is no longer used by StorageObserver)
func (*StorageObserver) Listen() {}

// Stop to satisfy Listener interface (but is no longer used by StorageObserver)
func (s *StorageObserver) Stop() error {
	s.wsconn.StorageAPI.UnregisterStorageObserver(s)
	return nil
}

// BlockListener to handle listening for blocks importedChan
type BlockListener struct {
	Channel       chan *types.Block
	wsconn        *WSConn
	subID         uint32
	done          chan struct{}
	cancel        chan struct{}
	cancelTimeout time.Duration
}

// NewBlockListener constructor for creating BlockListener
func NewBlockListener(conn *WSConn) *BlockListener {
	bl := &BlockListener{
		wsconn:        conn,
		cancel:        make(chan struct{}, 1),
		cancelTimeout: defaultCancelTimeout,
		done:          make(chan struct{}, 1),
	}
	return bl
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockListener) Listen() {
	go func() {
		defer func() {
			l.wsconn.BlockAPI.FreeImportedBlockNotifierChannel(l.Channel)
			close(l.done)
		}()

		for {
			select {
			case <-l.cancel:
				return
			case block, ok := <-l.Channel:
				if !ok {
					return
				}

				if block == nil {
					continue
				}
				head, err := modules.HeaderToJSON(block.Header)
				if err != nil {
					logger.Error("failed to convert header to JSON", "error", err)
				}

				res := newSubcriptionBaseResponseJSON()
				res.Method = chainNewHeadMethod
				res.Params.Result = head
				res.Params.SubscriptionID = l.subID
				l.wsconn.safeSend(res)
			}
		}
	}()
}

// Stop to cancel the running goroutines to this listener
func (l *BlockListener) Stop() error {
	return cancelWithTimeout(l.cancel, l.done, l.cancelTimeout)
}

// BlockFinalizedListener to handle listening for finalised blocks
type BlockFinalizedListener struct {
	channel       chan *types.FinalisationInfo
	wsconn        *WSConn
	subID         uint32
	done          chan struct{}
	cancel        chan struct{}
	cancelTimeout time.Duration
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *BlockFinalizedListener) Listen() {
	go func() {
		defer func() {
			l.wsconn.BlockAPI.FreeFinalisedNotifierChannel(l.channel)
			close(l.done)
		}()

		for {
			select {
			case <-l.cancel:
				return
			case info, ok := <-l.channel:
				if !ok {
					return
				}

				if info == nil {
					continue
				}
				head, err := modules.HeaderToJSON(info.Header)
				if err != nil {
					logger.Error("failed to convert header to JSON", "error", err)
				}
				res := newSubcriptionBaseResponseJSON()
				res.Method = chainFinalizedHeadMethod
				res.Params.Result = head
				res.Params.SubscriptionID = l.subID
				l.wsconn.safeSend(res)
			}
		}
	}()
}

// Stop to cancel the running goroutines to this listener
func (l *BlockFinalizedListener) Stop() error {
	return cancelWithTimeout(l.cancel, l.done, l.cancelTimeout)
}

// AllBlocksListener is a listener that is aware of new and newly finalised blocks```
type AllBlocksListener struct {
	finalizedChan chan *types.FinalisationInfo
	importedChan  chan *types.Block

	wsconn        *WSConn
	subID         uint32
	done          chan struct{}
	cancel        chan struct{}
	cancelTimeout time.Duration
}

func newAllBlockListener(conn *WSConn) *AllBlocksListener {
	return &AllBlocksListener{
		cancel:        make(chan struct{}, 1),
		done:          make(chan struct{}, 1),
		cancelTimeout: defaultCancelTimeout,
		wsconn:        conn,
	}
}

// Listen start a goroutine to listen imported and finalised blocks
func (l *AllBlocksListener) Listen() {
	go func() {
		defer func() {
			l.wsconn.BlockAPI.FreeImportedBlockNotifierChannel(l.importedChan)
			l.wsconn.BlockAPI.FreeFinalisedNotifierChannel(l.finalizedChan)

			close(l.done)
		}()

		for {
			select {
			case <-l.cancel:
				return
			case fin, ok := <-l.finalizedChan:
				if !ok {
					return
				}

				if fin == nil {
					continue
				}

				finHead, err := modules.HeaderToJSON(fin.Header)
				if err != nil {
					logger.Error("failed to convert finalised block header to JSON", "error", err)
					continue
				}

				l.wsconn.safeSend(newSubscriptionResponse(chainAllHeadMethod, l.subID, finHead))

			case imp, ok := <-l.importedChan:
				if !ok {
					return
				}

				if imp == nil {
					continue
				}

				impHead, err := modules.HeaderToJSON(imp.Header)
				if err != nil {
					logger.Error("failed to convert imported block header to JSON", "error", err)
					continue
				}

				l.wsconn.safeSend(newSubscriptionResponse(chainAllHeadMethod, l.subID, impHead))
			}
		}
	}()
}

// Stop will unregister the imported chanells and stop the goroutine
func (l *AllBlocksListener) Stop() error {
	return cancelWithTimeout(l.cancel, l.done, l.cancelTimeout)
}

// ExtrinsicSubmitListener to handle listening for extrinsic events
type ExtrinsicSubmitListener struct {
	wsconn        *WSConn
	subID         uint32
	extrinsic     types.Extrinsic
	importedChan  chan *types.Block
	importedHash  common.Hash
	finalisedChan chan *types.FinalisationInfo
	done          chan struct{}
	cancel        chan struct{}
	cancelTimeout time.Duration
}

// NewExtrinsicSubmitListener constructor to build new ExtrinsicSubmitListener
func NewExtrinsicSubmitListener(conn *WSConn, extBytes []byte) *ExtrinsicSubmitListener {
	esl := &ExtrinsicSubmitListener{
		wsconn:        conn,
		extrinsic:     types.Extrinsic(extBytes),
		finalisedChan: make(chan *types.FinalisationInfo),
		cancel:        make(chan struct{}, 1),
		done:          make(chan struct{}, 1),
		cancelTimeout: defaultCancelTimeout,
	}
	return esl
}

// Listen implementation of Listen interface to listen for importedChan changes
func (l *ExtrinsicSubmitListener) Listen() {
	// listen for imported blocks with extrinsic
	go func() {
		defer func() {
			l.wsconn.BlockAPI.FreeImportedBlockNotifierChannel(l.importedChan)
			l.wsconn.BlockAPI.FreeFinalisedNotifierChannel(l.finalisedChan)
			close(l.done)
			close(l.finalisedChan)
		}()

		for {
			select {
			case <-l.cancel:
				return
			case block, ok := <-l.importedChan:
				if !ok {
					return
				}

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
					l.wsconn.safeSend(newSubscriptionResponse(authorExtrinsicUpdatesMethod, l.subID, resM))
				}

			case info, ok := <-l.finalisedChan:
				if !ok {
					return
				}

				if reflect.DeepEqual(l.importedHash, info.Header.Hash()) {
					resM := make(map[string]interface{})
					resM["finalised"] = info.Header.Hash().String()
					l.wsconn.safeSend(newSubscriptionResponse(authorExtrinsicUpdatesMethod, l.subID, resM))
				}
			}
		}
	}()
}

// Stop to cancel the running goroutines to this listener
func (l *ExtrinsicSubmitListener) Stop() error {
	return cancelWithTimeout(l.cancel, l.done, l.cancelTimeout)
}

// RuntimeVersionListener to handle listening for Runtime Version
type RuntimeVersionListener struct {
	wsconn        WSConnAPI
	subID         uint32
	runtimeUpdate chan runtime.Version
	channelID     uint32
	coreAPI       modules.CoreAPI
}

// VersionListener interface defining methods that version listener must implement
type VersionListener interface {
	GetChannelID() uint32
}

// Listen implementation of Listen interface to listen for runtime version changes
func (l *RuntimeVersionListener) Listen() {
	// This sends current runtime version once when subscription is created
	rtVersion, err := l.coreAPI.GetRuntimeVersion(nil)
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

	go l.wsconn.safeSend(newSubscriptionResponse(stateRuntimeVersionMethod, l.subID, ver))

	// listen for runtime updates
	go func() {
		for {
			info, ok := <-l.runtimeUpdate
			if !ok {
				return
			}

			ver := modules.StateRuntimeVersionResponse{}

			ver.SpecName = string(info.SpecName())
			ver.ImplName = string(info.ImplName())
			ver.AuthoringVersion = info.AuthoringVersion()
			ver.SpecVersion = info.SpecVersion()
			ver.ImplVersion = info.ImplVersion()
			ver.TransactionVersion = info.TransactionVersion()
			ver.Apis = modules.ConvertAPIs(info.APIItems())

			l.wsconn.safeSend(newSubscriptionResponse(stateRuntimeVersionMethod, l.subID, ver))
		}
	}()
}

// GetChannelID function that returns listener's channel ID
func (l *RuntimeVersionListener) GetChannelID() uint32 {
	return l.channelID
}

// Stop to runtimeVersionListener not implemented yet because the listener
// does not need to be stoped
func (*RuntimeVersionListener) Stop() error { return nil }

// GrandpaJustificationListener struct has the finalisedCh and the context to stop the goroutines
type GrandpaJustificationListener struct {
	cancel        chan struct{}
	cancelTimeout time.Duration
	done          chan struct{}
	wsconn        *WSConn
	subID         uint32
	finalisedCh   chan *types.FinalisationInfo
}

// Listen will start goroutines that listen to the finaised blocks
func (g *GrandpaJustificationListener) Listen() {
	// listen for finalised headers
	go func() {
		defer func() {
			g.wsconn.BlockAPI.FreeFinalisedNotifierChannel(g.finalisedCh)
			close(g.done)
		}()

		for {
			select {
			case <-g.cancel:
				return

			case info, ok := <-g.finalisedCh:
				if !ok {
					return
				}

				just, err := g.wsconn.BlockAPI.GetJustification(info.Header.Hash())
				if err != nil {
					g.wsconn.safeSendError(float64(g.subID), big.NewInt(InvalidRequestCode),
						fmt.Sprintf("failed to retrieve justification: %v", err))
				}

				g.wsconn.safeSend(newSubscriptionResponse(grandpaJustificationsMethod, g.subID, common.BytesToHex(just)))
			}
		}
	}()
}

// Stop will cancel all the goroutines that are executing
func (g *GrandpaJustificationListener) Stop() error {
	return cancelWithTimeout(g.cancel, g.done, g.cancelTimeout)
}

func cancelWithTimeout(cancel, done chan struct{}, t time.Duration) error {
	close(cancel)

	timeout := time.NewTimer(t)
	defer timeout.Stop()

	select {
	case <-done:
		return nil
	case <-timeout.C:
		return ErrCannotCancel
	}
}
