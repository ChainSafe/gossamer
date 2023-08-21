package parachain

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
)

type Context struct {
	Sender   Sender
	Receiver chan any
}

type Sender interface {
	SendMessage(msg any) error
	Feed(msg any) error
}

type ActivatedLeaf struct {
	Hash   common.Hash
	Number uint32
}

type Overseer struct {
	receiverChan      chan any
	subsystemContexts map[Subsystem]Context
}

func NewOverseer() *Overseer {
	overseer := &Overseer{
		receiverChan:      make(chan any),
		subsystemContexts: make(map[Subsystem]Context),
	}
	return overseer
}

func (o *Overseer) start() {
	for subsystem, context := range o.subsystemContexts {
		go subsystem.Run(context)
	}

	go func() {
		for {
			select {
			case msg := <-o.receiverChan:
				fmt.Printf("overseer received msg %v\n", msg)
			}
		}
	}()
	fmt.Printf("Overseer started\n")
}
func (o *Overseer) stop() {
	fmt.Printf("Overseer stopped\n")
}

func (o *Overseer) sendActiveLeaf() {
	for _, context := range o.subsystemContexts {
		context.Receiver <- &ActivatedLeaf{
			Hash:   common.Hash{0, 1, 2, 3},
			Number: 4,
		}
	}
}

type messageType int

const (
	Message1 messageType = iota
	Message2
)

type SubsystemMessage struct {
	Sender   Subsystem
	Receiver Subsystem
	msgType  messageType
	content  interface{}
}

type Subsystem interface {
	Run(context Context) error
}

type ExampleSender struct {
	senderChan chan any
}

func (e *ExampleSender) SendMessage(msg any) error {
	e.senderChan <- msg
	return nil
}
func (e *ExampleSender) Feed(msg any) error {
	fmt.Printf("feed message %v\n", msg)
	return nil
}

func (o *Overseer) RegisterSubSystem(subsystem Subsystem) {
	sender := &ExampleSender{
		senderChan: o.receiverChan,
	}
	context := Context{
		Sender:   sender,
		Receiver: make(chan any),
	}
	o.subsystemContexts[subsystem] = context
}
