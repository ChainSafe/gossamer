package parachain

import (
	"fmt"
	"testing"
	"time"
)

type ExampleSubsystem1 struct {
	name string
}

func (e *ExampleSubsystem1) Start() {
	fmt.Printf("%v started\n", e.name)
}
func (e *ExampleSubsystem1) Stop() {
	fmt.Printf("%v stopped\n", e.name)
}
func (e *ExampleSubsystem1) SendMessage(message SubsystemMessage) {
}

func (e *ExampleSubsystem1) ProcessMessage() {
}

func (e *ExampleSubsystem1) ReceiveMessage() SubsystemMessage {
	return SubsystemMessage{}
}

func TestStartSubsystems(t *testing.T) {
	overseer := NewOverseer()

	ss1 := &ExampleSubsystem1{
		name: "subSystem 1",
	}
	overseer.RegisterSubSystem(ss1)
	overseer.start()
	overseer.startSubsystems()

	time.Sleep(5 * time.Second)
	overseer.stopSubsystems()
	overseer.stop()
}
