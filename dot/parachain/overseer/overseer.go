package overseer

import (
	"fmt"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"time"
)

type Overseer struct {
	context    *Context
	subsystems []Subsystem
}

func NewOverseer() *Overseer {
	comChan := make(chan any)
	overseer := &Overseer{
		context: &Context{
			Sender:   &ExampleSender{senderChan: comChan},
			Receiver: comChan,
		},
	}
	return overseer
}

func (o *Overseer) Start() {
	//for subsystem, context := range o.subsystemContexts {
	//	go subsystem.Run(context)
	//}

	go func() {
		for {
			select {
			case msg := <-o.context.Receiver:
				fmt.Printf("overseer received msg: %T, %v\n", msg, msg)
				switch message := msg.(type) {
				case ChainAPIMessage:
					response := uint32(1)
					message.ResponseChannel <- &response
				case AvailabilityRecoveryMessage:
					response := RecoveryErrorUnavailable
					message.ResponseChannel <- AvailabilityRecoveryResponse{
						Error: &response,
					}
				default:
					fmt.Printf("unknown message type %T\n", msg)
				}
			}
		}
	}()
	fmt.Printf("Overseer started\n")

	// send test ActiveLeaf, normally this would be sent by the parachain
	time.Sleep(time.Millisecond * 500)
	err := o.sendActiveLeaf(parachainTypes.BlockNumber(11))
	if err != nil {
		panic(err)
	}
}
func (o *Overseer) stop() {
	fmt.Printf("Overseer stopped\n")
}

func (o *Overseer) sendActiveLeaf(blockNumber parachainTypes.BlockNumber) error {
	encodedBlockNumber, err := blockNumber.Encode()
	if err != nil {
		return fmt.Errorf("failed to encode block number: %w", err)
	}
	parentHash, err := common.Blake2bHash(encodedBlockNumber)
	if err != nil {
		return fmt.Errorf("failed to hash block number: %w", err)
	}

	blockHeader := types.Header{
		ParentHash:     parentHash,
		Number:         uint(blockNumber),
		StateRoot:      common.Hash{},
		ExtrinsicsRoot: common.Hash{},
		Digest:         scale.VaryingDataTypeSlice{},
	}
	blockHash := blockHeader.Hash()

	update := ActiveLeavesUpdate{
		Activated: &ActivatedLeaf{
			Hash:   blockHash,
			Number: uint32(blockNumber),
		},
	}

	for _, subsystem := range o.subsystems {
		subsystem.ProcessActiveLeavesUpdate(update)
	}
	return nil
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
	//Run(context Context) error

	// ProcessActiveLeavesUpdate processes an active leaves update
	ProcessActiveLeavesUpdate(update ActiveLeavesUpdate) error
}

type ExampleSender struct {
	senderChan chan any
}

func NewExampleSender() *ExampleSender {
	return &ExampleSender{}
}

func (e *ExampleSender) SendMessage(msg any) error {
	e.senderChan <- msg
	return nil
}
func (e *ExampleSender) Feed(msg any) error {
	fmt.Printf("feed message %v\n", msg)
	return nil
}

func (o *Overseer) GetContext() *Context {
	return o.context
}

func (o *Overseer) RegisterSubSystem(subsystem Subsystem) {
	o.subsystems = append(o.subsystems, subsystem)
}
