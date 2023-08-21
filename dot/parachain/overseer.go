package parachain

import "fmt"

type Overseer struct {
	subsystems []Subsystem
	channels   map[Subsystem]chan SubsystemMessage
}

func NewOverseer() *Overseer {
	overseer := &Overseer{
		subsystems: nil,
		channels:   make(map[Subsystem]chan SubsystemMessage),
	}
	return overseer
}

func (o *Overseer) start() {
	fmt.Printf("Overseer started\n")
}
func (o *Overseer) stop() {
	fmt.Printf("Overseer stopped\n")
}
func (o *Overseer) startSubsystems() {
	for _, subsystem := range o.subsystems {
		subsystem.Start()
	}
}

func (o *Overseer) stopSubsystems() {
	for _, subsystem := range o.subsystems {
		subsystem.Stop()
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
	Start()
	Stop()
	SendMessage(message SubsystemMessage)
	ProcessMessage()
	ReceiveMessage() SubsystemMessage // consider chan here for receiving
}

func (o *Overseer) RegisterSubSystem(subsystem Subsystem) {
	o.subsystems = append(o.subsystems, subsystem)
	//todo: create chan for subsystems here (test)
	// store map of channels
	channel := make(chan SubsystemMessage)
	o.channels[subsystem] = channel
	go func() {
		for {
			select {
			case msg := <-channel:
				relayChan := o.channels[msg.Receiver]
				relayChan <- msg
			}
		}
	}()
}
