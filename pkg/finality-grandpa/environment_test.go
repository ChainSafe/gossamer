// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"
	"time"

	rand "math/rand/v2"
)

type ID uint32

type Signature uint32
type listenerItem struct {
	Hash   string
	Number uint32
	Commit Commit[string, uint32, Signature, ID]
}

type environment struct {
	chain                    *dummyChain
	localID                  ID
	network                  *Network
	listeners                []chan listenerItem
	lastCompleteAndConcluded [2]uint64
	// roundIn holds the inbound channels handed to the voter, per round, so they
	// can be closed once the round concludes. RoundData is called more than once
	// for a round number, hence a slice.
	roundIn map[uint64][]chan SignedMessageError[string, uint32, Signature, ID]
	mtx     sync.Mutex

	concludedCalled chan struct{}
}

func newEnvironment(network *Network, localID ID) environment {
	return environment{
		chain:           newDummyChain(),
		localID:         localID,
		network:         network,
		roundIn:         make(map[uint64][]chan SignedMessageError[string, uint32, Signature, ID]),
		concludedCalled: make(chan struct{}),
	}
}

func (e *environment) WithChain(f func(*dummyChain)) {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	f(e.chain)
}

func (e *environment) FinalizedStream() chan listenerItem {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	ch := make(chan listenerItem)
	e.listeners = append(e.listeners, ch)
	return ch
}

func (e *environment) LastCompletedAndConcluded() [2]uint64 {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	return e.lastCompleteAndConcluded
}

func (e *environment) Ancestry(base, block string) (ancestors []string, err error) {
	return e.chain.Ancestry(base, block)
}

func (e *environment) IsEqualOrDescendantOf(base, block string) bool {
	return e.chain.IsEqualOrDescendantOf(base, block)
}

func (e *environment) BestChainContaining(base string) BestChain[string, uint32] {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	ch := make(chan BestChainOutput[string, uint32], 1)
	ch <- BestChainOutput[string, uint32]{Value: e.chain.BestChainContaining(base)}
	close(ch)
	return ch
}

func (e *environment) RoundData(
	round uint64,
) RoundData[string, uint32, Signature, ID, Message[string, uint32]] {
	outgoing := make(Output[string, uint32])
	incoming := e.network.MakeRoundComms(round, e.localID, outgoing)

	// Remember it so Concluded can close it: the voter reads this channel through
	// a forwarding goroutine that ends only when the channel does.
	e.mtx.Lock()
	e.roundIn[round] = append(e.roundIn[round], incoming)
	e.mtx.Unlock()

	var outgoingFunc = func(m Message[string, uint32]) error {
		outgoing <- m
		return nil
	}

	rd := RoundData[string, uint32, Signature, ID, Message[string, uint32]]{
		VoterID:        &e.localID,
		PrevoteTimer:   *time.NewTimer(500 * time.Millisecond),
		PrecommitTimer: *time.NewTimer(1000 * time.Millisecond),
		Incoming:       incoming,
		Outgoing:       outgoingFunc,
	}
	return rd
}

func (*environment) RoundCommitTimer() time.Timer {
	timer := time.NewTimer(time.Duration(rand.Int64N(1000)) * time.Millisecond)
	return *timer
}

func (e *environment) Completed(
	round uint64,
	_ RoundState[string, uint32],
	_ HashNumber[string, uint32],
	_ HistoricalVotes[string, uint32, Signature, ID],
) error {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	e.lastCompleteAndConcluded[0] = round
	return nil
}

func (e *environment) Concluded(
	round uint64,
	_ RoundState[string, uint32],
	_ HashNumber[string, uint32],
	_ HistoricalVotes[string, uint32, Signature, ID],
) error {
	e.mtx.Lock()
	e.lastCompleteAndConcluded[1] = round
	incoming := e.roundIn[round]
	delete(e.roundIn, round)
	e.mtx.Unlock()

	// The round is over, so release the inbound channels handed out for it.
	for _, in := range incoming {
		e.network.StopRoundComms(round, in)
	}

	go func() {
		e.concludedCalled <- struct{}{}
	}()
	return nil
}

func (e *environment) FinalizeBlock(
	hash string,
	number uint32,
	_ uint64,
	commit Commit[string, uint32, Signature, ID],
) error {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	lastFinalizedHash, lastFinalizedNumber := e.chain.LastFinalized()
	if number <= lastFinalizedNumber {
		panic("Attempted to finalize backwards")
	}

	if _, err := e.chain.Ancestry(lastFinalizedHash, hash); err != nil {
		panic("Safety violation: reverting finalized block.")
	}

	e.chain.SetLastFinalized(hash, number)
	for _, listener := range e.listeners {
		listener <- listenerItem{
			hash, number, commit,
		}
	}
	return nil
}

func (*environment) Proposed(_ uint64, _ PrimaryPropose[string, uint32]) error {
	return nil
}

func (*environment) Prevoted(_ uint64, _ Prevote[string, uint32]) error {
	return nil
}

func (*environment) Precommitted(_ uint64, _ Precommit[string, uint32]) error {
	return nil
}

func (*environment) PrevoteEquivocation(
	round uint64,
	equivocation Equivocation[ID, Prevote[string, uint32], Signature],
) {
	panic(fmt.Errorf("Encountered equivocation in round %v: %v", round, equivocation))
}

// Note that an equivocation in prevotes has occurred.
func (*environment) PrecommitEquivocation(
	round uint64,
	equivocation Equivocation[ID, Precommit[string, uint32], Signature],
) {
	panic(fmt.Errorf("Encountered equivocation in round %v: %v", round, equivocation))
}

// p2p network data for a round.
type BroadcastNetwork[M, N any] struct {
	receiver    chan M
	stop        chan struct{}
	mu          sync.Mutex
	senders     []chan M
	history     []M
	routing     bool
	stopped     bool
	routeWG     sync.WaitGroup
	forwarderWG sync.WaitGroup
}

func NewBroadcastNetwork[M, N any]() *BroadcastNetwork[M, N] {
	bn := BroadcastNetwork[M, N]{
		receiver: make(chan M, 10000),
		stop:     make(chan struct{}),
	}
	return &bn
}

func (bm *BroadcastNetwork[M, N]) SendMessage(message M) {
	select {
	case bm.receiver <- message:
	case <-bm.stop:
	}
}

func (bm *BroadcastNetwork[M, N]) AddNode(f func(N) M, out chan N) (in chan M) {
	// buffer to 100 messages for now
	in = make(chan M, 10000)

	bm.mu.Lock()
	// get history to the node.
	for _, priorMessage := range bm.history {
		in <- priorMessage
	}
	bm.senders = append(bm.senders, in)
	startRoute := !bm.routing
	if startRoute {
		bm.routing = true
		bm.routeWG.Add(1)
	}
	bm.mu.Unlock()

	if startRoute {
		go bm.route()
	}

	bm.forwarderWG.Add(1)
	go func() {
		defer bm.forwarderWG.Done()
		for {
			select {
			case n, ok := <-out:
				if !ok {
					return
				}
				select {
				case bm.receiver <- f(n):
				case <-bm.stop:
					return
				}
			case <-bm.stop:
				return
			}
		}
	}()
	return in
}

func (bm *BroadcastNetwork[M, N]) route() {
	defer bm.routeWG.Done()
	for msg := range bm.receiver {
		// Under the lock: RemoveNode closes a node's channel, and closing one a
		// producer is about to send on panics. Senders are buffered, so holding it
		// across the delivery does not block.
		bm.mu.Lock()
		bm.history = append(bm.history, msg)
		for _, sender := range bm.senders {
			sender <- msg
		}
		bm.mu.Unlock()
	}
}

// RemoveNode deregisters a node's inbound channel and closes it, shutting down
// the voter reading it. Held under bm.mu so it cannot race a delivery in route.
func (bm *BroadcastNetwork[M, N]) RemoveNode(in chan M) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	for i, sender := range bm.senders {
		if sender == in {
			bm.senders = append(bm.senders[:i], bm.senders[i+1:]...)
			close(in)
			return
		}
	}
}

func (bm *BroadcastNetwork[M, N]) Stop() {
	bm.mu.Lock()
	if bm.stopped {
		bm.mu.Unlock()
		return
	}
	bm.stopped = true
	close(bm.stop)
	bm.mu.Unlock()

	// Order matters: drain forwarders first so they stop sending into receiver,
	// then close receiver so route can exit, then close per-node senders.
	bm.forwarderWG.Wait()
	close(bm.receiver)
	bm.routeWG.Wait()
	bm.mu.Lock()
	senders := bm.senders
	bm.senders = nil
	bm.mu.Unlock()
	for _, sender := range senders {
		close(sender)
	}
}

type RoundNetwork struct {
	*BroadcastNetwork[SignedMessageError[string, uint32, Signature, ID], Message[string, uint32]]
}

func NewRoundNetwork() *RoundNetwork {
	bn := NewBroadcastNetwork[SignedMessageError[string, uint32, Signature, ID], Message[string, uint32]]()
	rn := RoundNetwork{bn}
	return &rn
}

func (rn *RoundNetwork) AddNode(
	f func(Message[string, uint32]) SignedMessageError[string, uint32, Signature, ID],
	out chan Message[string, uint32],
) (in chan SignedMessageError[string, uint32, Signature, ID]) {
	return rn.BroadcastNetwork.AddNode(f, out)
}

type GlobalMessageNetwork struct {
	*BroadcastNetwork[GlobalInItem[string, uint32, Signature, ID], CommunicationOut[string, uint32, Signature, ID]]
}

func NewGlobalMessageNetwork() *GlobalMessageNetwork {
	bn := NewBroadcastNetwork[GlobalInItem[
		string, uint32, Signature, ID], CommunicationOut[string, uint32, Signature, ID],
	]()
	gmn := GlobalMessageNetwork{bn}
	return &gmn
}

func (gmn *GlobalMessageNetwork) AddNode(
	f func(CommunicationOut[string, uint32, Signature, ID]) GlobalInItem[string, uint32, Signature, ID],
	out chan CommunicationOut[string, uint32, Signature, ID],
) (in chan GlobalInItem[string, uint32, Signature, ID]) {
	return gmn.BroadcastNetwork.AddNode(f, out)
}

// A test network. Instantiate this with `make_network`,
type Network struct {
	rounds         map[uint64]*RoundNetwork
	globalMessages GlobalMessageNetwork
	mtx            sync.Mutex
}

func NewNetwork() *Network {
	return &Network{
		rounds:         make(map[uint64]*RoundNetwork),
		globalMessages: *NewGlobalMessageNetwork(),
	}
}

func (n *Network) Stop() {
	for _, rn := range n.rounds {
		rn.Stop()
	}
	n.globalMessages.Stop()
}

func (n *Network) MakeRoundComms(
	roundNumber uint64,
	nodeID ID,
	out chan Message[string, uint32],
) (in chan SignedMessageError[string, uint32, Signature, ID]) {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	round, ok := n.rounds[roundNumber]
	if !ok {
		round = NewRoundNetwork()
		n.rounds[roundNumber] = round
	}
	return round.AddNode(func(message Message[string, uint32]) SignedMessageError[string, uint32, Signature, ID] {
		return SignedMessageError[string, uint32, Signature, ID]{
			SignedMessage: SignedMessage[string, uint32, Signature, ID]{
				Message:   message,
				Signature: Signature(nodeID),
				ID:        nodeID,
			},
		}
	}, out,
	)
}

func (n *Network) MakeGlobalComms(
	out chan CommunicationOut[string, uint32, Signature, ID],
) chan GlobalInItem[string, uint32, Signature, ID] {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	return n.globalMessages.AddNode(
		func(message CommunicationOut[string, uint32, Signature, ID]) GlobalInItem[string, uint32, Signature, ID] {
			if message == nil {
				panic("nil message variant")
			}
			switch message := message.(type) {
			case CommunicationOutCommit[string, uint32, Signature, ID]:
				ci := CommunicationInCommit[string, uint32, Signature, ID]{
					Number:        message.Number,
					CompactCommit: message.Commit.CompactCommit(),
					Callback:      nil,
				}
				return GlobalInItem[string, uint32, Signature, ID]{
					CommunicationIn: ci,
				}
			default:
				panic("invalid CommunicationOut variant")
			}
		}, out)
}

// StopRoundComms closes one inbound channel handed out by MakeRoundComms. Only
// that node's channel: the round network is shared, and other voters may still
// be in this round.
func (n *Network) StopRoundComms(
	roundNumber uint64,
	in chan SignedMessageError[string, uint32, Signature, ID],
) {
	n.mtx.Lock()
	round, ok := n.rounds[roundNumber]
	n.mtx.Unlock()

	if ok {
		round.RemoveNode(in)
	}
}

// StopGlobalComms closes the inbound channel handed to a voter by
// MakeGlobalComms, which is how that voter is shut down.
func (n *Network) StopGlobalComms(in chan GlobalInItem[string, uint32, Signature, ID]) {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	n.globalMessages.RemoveNode(in)
}

func (n *Network) SendMessage(message CommunicationIn[string, uint32, Signature, ID]) {
	n.globalMessages.SendMessage(GlobalInItem[string, uint32, Signature, ID]{message, nil})
}
