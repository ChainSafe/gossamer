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

type timer struct {
	wakerChan *wakerChan[error]
	mtx       sync.Mutex
	expired   bool
}

func newTimer(in <-chan time.Time) *timer {
	inErr := make(chan error)
	wc := newWakerChan(inErr)
	t := timer{wakerChan: wc}
	go t.poll(in)
	return &t
}

func (t *timer) poll(in <-chan time.Time) {
	<-in
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.wakerChan.in != nil {
		t.wakerChan.in <- nil
		close(t.wakerChan.in)
		t.wakerChan.in = nil
	}
	t.expired = true
}

func (t *timer) SetWaker(waker *waker) {
	t.wakerChan.setWaker(waker)
}

func (t *timer) Elapsed() (bool, error) {
	return t.expired, nil
}

func (t *timer) Close() {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.wakerChan.in != nil {
		close(t.wakerChan.in)
		t.wakerChan.in = nil
	}
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

	concludedCalled chan struct{}
}

func newEnvironment(network *Network, localID ID) environment {
	return environment{
		chain:           newDummyChain(),
		localID:         localID,
		network:         network,
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
	outgoing Output[string, uint32],
) RoundData[string, uint32, Signature, ID] {
	incoming := e.network.MakeRoundComms(round, e.localID, outgoing)

	rd := RoundData[string, uint32, Signature, ID]{
		VoterID:        &e.localID,
		PrevoteTimer:   newTimer(time.NewTimer(500 * time.Millisecond).C),
		PrecommitTimer: newTimer(time.NewTimer(1000 * time.Millisecond).C),
		Incoming:       incoming,
	}
	return rd
}

func (*environment) RoundCommitTimer() Timer {
	inner := time.NewTimer(time.Duration(rand.Int64N(1000)) * time.Millisecond).C
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
	receiver chan M
	senders  []chan M
	history  []M
	routing  bool
	wg       sync.WaitGroup
}

func NewBroadcastNetwork[M, N any]() *BroadcastNetwork[M, N] {
	bn := BroadcastNetwork[M, N]{
		receiver: make(chan M, 10000),
	}
	return &bn
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
		bm.wg.Add(1)
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
	defer bm.wg.Done()
	for msg := range bm.receiver {
		bm.history = append(bm.history, msg)
		for _, sender := range bm.senders {
			sender <- msg
		}
	}
}

func (bm *BroadcastNetwork[M, N]) Stop() {
	close(bm.receiver)
	bm.wg.Wait()
	for _, sender := range bm.senders {
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
	*BroadcastNetwork[globalInItem[string, uint32, Signature, ID], CommunicationOut[string, uint32, Signature, ID]]
}

func NewGlobalMessageNetwork() *GlobalMessageNetwork {
	bn := NewBroadcastNetwork[globalInItem[
		string, uint32, Signature, ID], CommunicationOut[string, uint32, Signature, ID],
	]()
	gmn := GlobalMessageNetwork{bn}
	return &gmn
}

func (gmn *GlobalMessageNetwork) AddNode(
	f func(CommunicationOut[string, uint32, Signature, ID]) globalInItem[string, uint32, Signature, ID],
	out chan CommunicationOut[string, uint32, Signature, ID],
) (in chan globalInItem[string, uint32, Signature, ID]) {
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
) chan globalInItem[string, uint32, Signature, ID] {
	n.mtx.Lock()
	defer n.mtx.Unlock()

	return n.globalMessages.AddNode(
		func(message CommunicationOut[string, uint32, Signature, ID]) globalInItem[string, uint32, Signature, ID] {
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
				return globalInItem[string, uint32, Signature, ID]{
					CommunicationIn: ci,
				}
			default:
				panic("invalid CommunicationOut variant")
			}
		}, out)
}

func (n *Network) SendMessage(message CommunicationIn[string, uint32, Signature, ID]) {
	n.globalMessages.SendMessage(globalInItem[string, uint32, Signature, ID]{message, nil})
}
