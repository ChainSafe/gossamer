// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"fmt"
	"sync"
	"time"

	availability_store "github.com/ChainSafe/gossamer/dot/parachain/availability-store"
	"github.com/ChainSafe/gossamer/dot/parachain/backing"
	"github.com/ChainSafe/gossamer/dot/parachain/chainapi"
	collatorprotocolmessages "github.com/ChainSafe/gossamer/dot/parachain/collator-protocol/messages"
	networkbridgemessages "github.com/ChainSafe/gossamer/dot/parachain/network-bridge/messages"
	provisionermessages "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	statementedistributionmessages "github.com/ChainSafe/gossamer/dot/parachain/statement-distribution/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

var (
	logger = log.NewFromGlobal(log.AddContext("pkg", "parachain-overseer"))
)

type Overseer interface {
	Start() error
	RegisterSubsystem(subsystem parachaintypes.Subsystem)
	Stop() error
	GetSubsystemToOverseerChannel() chan any
}

type OverseerSystem struct {
	ctx     context.Context
	cancel  context.CancelFunc
	errChan chan error // channel for overseer to send errors to service that started it

	blockState   BlockState
	activeLeaves map[common.Hash]uint32

	// block notification channels
	imported  chan *types.Block
	finalised chan *types.FinalisationInfo

	SubsystemsToOverseer chan any
	subsystems           map[parachaintypes.Subsystem]chan any // map[Subsystem]OverseerToSubSystem channel
	nameToSubsystem      map[parachaintypes.SubSystemName]parachaintypes.Subsystem
	wg                   sync.WaitGroup
}

// BlockState interface for block state methods
type BlockState interface {
	GetImportedBlockNotifierChannel() chan *types.Block
	FreeImportedBlockNotifierChannel(ch chan *types.Block)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	FreeFinalisedNotifierChannel(ch chan *types.FinalisationInfo)
	GetRuntime(hash common.Hash) (runtime.Instance, error)
}

func NewOverseer(blockState BlockState) *OverseerSystem {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	return &OverseerSystem{
		ctx:                  ctx,
		cancel:               cancel,
		errChan:              make(chan error),
		blockState:           blockState,
		activeLeaves:         make(map[common.Hash]uint32),
		SubsystemsToOverseer: make(chan any),
		subsystems:           make(map[parachaintypes.Subsystem]chan any),
		nameToSubsystem:      make(map[parachaintypes.SubSystemName]parachaintypes.Subsystem),
	}
}

func (o *OverseerSystem) GetSubsystemToOverseerChannel() chan any {
	return o.SubsystemsToOverseer
}

// RegisterSubsystem registers a subsystem with the overseer,
// Add OverseerToSubSystem channel to subsystem, which will be passed to subsystem's Run method.
func (o *OverseerSystem) RegisterSubsystem(subsystem parachaintypes.Subsystem) {
	OverseerToSubSystem := make(chan any)
	o.subsystems[subsystem] = OverseerToSubSystem
	o.nameToSubsystem[subsystem.Name()] = subsystem
}

func (o *OverseerSystem) Start() error {

	imported := o.blockState.GetImportedBlockNotifierChannel()
	finalised := o.blockState.GetFinalisedNotifierChannel()

	o.imported = imported
	o.finalised = finalised

	// start subsystems
	for subsystem, overseerToSubSystem := range o.subsystems {
		o.wg.Add(1)
		go func(sub parachaintypes.Subsystem, overseerToSubSystem chan any) {
			sub.Run(o.ctx, overseerToSubSystem)
			logger.Infof("subsystem %v stopped", sub)
			o.wg.Done()
		}(subsystem, overseerToSubSystem)
	}

	o.wg.Add(2)
	go o.processMessages()
	go o.handleBlockEvents()

	return nil
}

func (o *OverseerSystem) processMessages() {
	for {
		select {
		case msg := <-o.SubsystemsToOverseer:
			var subsystem parachaintypes.Subsystem

			switch msg := msg.(type) {
			case networkbridgemessages.DisconnectPeer, networkbridgemessages.ConnectToValidators,
				networkbridgemessages.ReportPeer, networkbridgemessages.SendCollationMessage,
				networkbridgemessages.SendValidationMessage:
				subsystem = o.nameToSubsystem[parachaintypes.NetworkBridgeSender]

			case backing.GetBackableCandidatesMessage, backing.CanSecondMessage, backing.SecondMessage, backing.StatementMessage:
				subsystem = o.nameToSubsystem[parachaintypes.CandidateBacking]

			case collatorprotocolmessages.CollateOn, collatorprotocolmessages.DistributeCollation,
				collatorprotocolmessages.ReportCollator, collatorprotocolmessages.Backed,
				collatorprotocolmessages.Invalid, collatorprotocolmessages.Seconded:

				subsystem = o.nameToSubsystem[parachaintypes.CollationProtocol]

			case availability_store.QueryAvailableData, availability_store.QueryDataAvailability,
				availability_store.QueryChunk, availability_store.QueryChunkSize, availability_store.QueryAllChunks,
				availability_store.QueryChunkAvailability, availability_store.StoreChunk,
				availability_store.StoreAvailableData:

				subsystem = o.nameToSubsystem[parachaintypes.AvailabilityStore]

			case statementedistributionmessages.Share, statementedistributionmessages.Backed:
				subsystem = o.nameToSubsystem[parachaintypes.StatementDistribution]

			case chainapi.ChainAPIMessage[util.Ancestors], chainapi.ChainAPIMessage[chainapi.BlockHeader]:
				subsystem = o.nameToSubsystem[parachaintypes.ChainAPI]
			case provisionermessages.RequestInherentData, provisionermessages.ProvisionableData:
				subsystem = o.nameToSubsystem[parachaintypes.Provisioner]

			case parachain.RuntimeAPIMessage:
				// TODO: this should be handled by the parachain runtime subsystem, see issue #3940
				rt, err := o.blockState.GetRuntime(msg.Hash)
				if err != nil {
					logger.Errorf("failed to get runtime: %v", err)
					continue
				}
				msg.Resp <- rt

			default:
				logger.Error("unknown message type")
			}

			overseerToSubsystem := o.subsystems[subsystem]
			overseerToSubsystem <- msg

		case <-o.ctx.Done():
			if err := o.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			o.wg.Done()
			return
		}
	}
}

func (o *OverseerSystem) handleBlockEvents() {
	for {
		select {
		case <-o.ctx.Done():
			if err := o.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			o.wg.Done()
			return
		case imported := <-o.imported:
			blockNumber, ok := o.activeLeaves[imported.Header.Hash()]
			if ok {
				if blockNumber != uint32(imported.Header.Number) {
					panic("block number mismatch")
				}
				return
			}

			o.activeLeaves[imported.Header.Hash()] = uint32(imported.Header.Number)
			delete(o.activeLeaves, imported.Header.ParentHash)

			// TODO:
			/*
				- Add active leaf only if given head supports parachain consensus.
				- You do that by checking the parachain host runtime api version.
				- If the parachain host runtime api version is at least 1, then the parachain consensus is supported.

					#[async_trait::async_trait]
					impl<Client> HeadSupportsParachains for Arc<Client>
					where
						Client: RuntimeApiSubsystemClient + Sync + Send,
					{
						async fn head_supports_parachains(&self, head: &Hash) -> bool {
							// Check that the `ParachainHost` runtime api is at least with version 1 present on chain.
							self.api_version_parachain_host(*head).await.ok().flatten().unwrap_or(0) >= 1
						}
					}

			*/
			activeLeavesUpdate := parachaintypes.ActiveLeavesUpdateSignal{
				Activated: &parachaintypes.ActivatedLeaf{
					Hash:   imported.Header.Hash(),
					Number: uint32(imported.Header.Number),
				},
				Deactivated: []common.Hash{imported.Header.ParentHash},
			}

			o.broadcast(activeLeavesUpdate)

		case finalised := <-o.finalised:
			deactivated := make([]common.Hash, 0)

			for hash, blockNumber := range o.activeLeaves {
				if blockNumber <= uint32(finalised.Header.Number) && hash != finalised.Header.Hash() {
					deactivated = append(deactivated, hash)
					delete(o.activeLeaves, hash)
				}
			}

			o.broadcast(parachaintypes.BlockFinalizedSignal{
				Hash:        finalised.Header.Hash(),
				BlockNumber: uint32(finalised.Header.Number),
			})

			// If there are no leaves being deactivated, we don't need to send an update.
			//
			// Our peers will be informed about our finalized block the next time we
			// activating/deactivating some leaf.
			if len(deactivated) > 0 {
				o.broadcast(parachaintypes.ActiveLeavesUpdateSignal{
					Deactivated: deactivated,
				})
			}
		}
	}
}

func (o *OverseerSystem) broadcast(msg any) {
	for _, overseerToSubSystem := range o.subsystems {
		overseerToSubSystem <- msg
	}
}

func (o *OverseerSystem) Stop() error {
	o.cancel()

	o.blockState.FreeImportedBlockNotifierChannel(o.imported)
	o.blockState.FreeFinalisedNotifierChannel(o.finalised)

	// close the errorChan to unblock any listeners on the errChan
	close(o.errChan)

	// wait for subsystems to stop
	// TODO: determine reasonable timeout duration for production, currently this is just for testing
	timedOut := waitTimeout(&o.wg, 500*time.Millisecond)
	fmt.Printf("timedOut: %v\n", timedOut)

	return nil
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) (timeouted bool) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	timeoutTimer := time.NewTimer(timeout)
	select {
	case <-c:
		if !timeoutTimer.Stop() {
			<-timeoutTimer.C
		}
		return false // completed normally
	case <-timeoutTimer.C:
		return true // timed out
	}
}
