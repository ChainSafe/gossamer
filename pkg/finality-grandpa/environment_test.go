// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

type ID uint32

type Signature uint32

type timer struct {
	wakerChan *wakerChan[error]
	expired   bool
}

func newTimer(in <-chan time.Time) *timer {
	inErr := make(chan error)
	wc := newWakerChan(inErr)
	t := timer{wakerChan: wc}
	go func() {
		<-in
		inErr <- nil
		t.expired = true
	}()
	return &t
}

func (t *timer) SetWaker(waker *waker) {
	t.wakerChan.SetWaker(waker)
}

func (t *timer) Elapsed() (bool, error) {
	return t.expired, nil
}

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
	mtx                      sync.Mutex
}

func newEnvironment(network *Network, localID ID) environment {
	return environment{
		chain:   newDummyChain(),
		localID: localID,
		network: network,
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
	return ch
}

func (e *environment) RoundData(
	round uint64,
	outgoing chan Message[string, uint32],
) RoundData[ID, Timer, SignedMessageError[string, uint32, Signature, ID]] {
	incoming := e.network.MakeRoundComms(round, e.localID, outgoing)

	rd := RoundData[ID, Timer, SignedMessageError[string, uint32, Signature, ID]]{
		VoterID:        &e.localID,
		PrevoteTimer:   newTimer(time.NewTimer(500 * time.Millisecond).C),
		PrecommitTimer: newTimer(time.NewTimer(1000 * time.Millisecond).C),
		Incoming:       incoming,
	}
	return rd
}

func (*environment) RoundCommitTimer() Timer {
	inner := time.NewTimer(time.Duration(rand.Int63n(1000)) * time.Millisecond).C
	timer := newTimer(inner)
	return timer
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
	defer e.mtx.Unlock()
	e.lastCompleteAndConcluded[1] = round
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
	receiver chan M
	senders  []chan M
	history  []M
	routing  bool
}

func NewBroadcastNetwork[M, N any]() BroadcastNetwork[M, N] {
	bn := BroadcastNetwork[M, N]{
		receiver: make(chan M, 10000),
	}
	return bn
}

func (bm *BroadcastNetwork[M, N]) SendMessage(message M) {
	bm.receiver <- message
}

func (bm *BroadcastNetwork[M, N]) AddNode(f func(N) M, out chan N) (in chan M) {
	// buffer to 100 messages for now
	in = make(chan M, 10000)

	// get history to the node.
	for _, priorMessage := range bm.history {
		in <- priorMessage
	}

	bm.senders = append(bm.senders, in)

	if !bm.routing {
		bm.routing = true
		go bm.route()
	}

	go func() {
		for n := range out {
			bm.receiver <- f(n)
		}
	}()
	return in
}

func (bm *BroadcastNetwork[M, N]) route() {
	for msg := range bm.receiver {
		bm.history = append(bm.history, msg)
		for _, sender := range bm.senders {
			sender <- msg
		}
	}
}

type RoundNetwork struct {
	BroadcastNetwork[SignedMessageError[string, uint32, Signature, ID], Message[string, uint32]]
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
	BroadcastNetwork[globalInItem, CommunicationOut]
}

func NewGlobalMessageNetwork() *GlobalMessageNetwork {
	bn := NewBroadcastNetwork[globalInItem, CommunicationOut]()
	gmn := GlobalMessageNetwork{bn}
	return &gmn
}

func (gmn *GlobalMessageNetwork) AddNode(
	f func(CommunicationOut) globalInItem,
	out chan CommunicationOut,
) (in chan globalInItem) {
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

func (n *Network) MakeGlobalComms(out chan CommunicationOut) chan globalInItem {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	return n.globalMessages.AddNode(func(message CommunicationOut) globalInItem {
		if message.variant == nil {
			panic("wtf?")
		}
		switch message := message.variant.(type) {
		case CommunicationOutCommit[string, uint32, Signature, ID]:
			ci := CommunicationIn{}
			setCommunicationIn[string, uint32, Signature, ID](&ci, CommunicationInCommit[string, uint32, Signature, ID]{
				Number:        message.Number,
				CompactCommit: message.Commit.CompactCommit(),
				Callback:      nil,
			})
			return globalInItem{
				CommunicationIn: ci,
			}
		default:
			panic("wtf")
		}
	}, out)
}

func (n *Network) SendMessage(message CommunicationIn) {
	n.globalMessages.SendMessage(globalInItem{message, nil})
}
