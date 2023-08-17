package parachain

type Overseer struct {
	//subsystems map[string]Subsystem
	channels map[Subsystem]chan SubsystemMessage
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
	SendMessage(message SubsystemMessage)
	ProcessMessage()
	ReceiveMessage() SubsystemMessage // consider chan here for receiving
}

func (o *Overseer) RegisterSubSystem(subsystem Subsystem) {
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
