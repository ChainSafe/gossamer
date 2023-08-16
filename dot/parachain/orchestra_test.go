package parachain

import (
	"fmt"
	"testing"
	"time"
)

type ExampleSubsystem struct {
	name string
	done chan struct{}
}

func NewExampleSubsystem(name string) *ExampleSubsystem {
	return &ExampleSubsystem{
		name: name,
		done: make(chan struct{}),
	}
}

func (s *ExampleSubsystem) Start(subCtx SubsystemContext) {
	fmt.Printf("Subsystem %s started\n", s.name)
	go func() {
		for {
			fmt.Printf("sub system %v start recieve\n", s.name)
			msg, err := subCtx.Recv()
			if err != nil {
				fmt.Printf("ERROR %v\n", err)
			}
			fmt.Printf("%v rec msg %v\n", s.name, msg)
		}

	}()
	// main loop for subsystem
	for {
		select {
		case <-s.done:
			fmt.Printf("Subsystem %s stopped\n", s.name)
			return
		default:
			subCtx.SendMessage(fmt.Sprintf("Subsystem %v working...", s.name))
			time.Sleep(time.Second)
		}
	}
}

func (s *ExampleSubsystem) Stop() {
	close(s.done)
}

type ExampleSubsystemContext struct {
	msgChan    chan interface{}
	recMsgChan chan interface{}
}

func (e *ExampleSubsystemContext) SendMessage(msg interface{}) {
	e.msgChan <- fmt.Sprintf("%v", msg)
}

func (e *ExampleSubsystemContext) Recv() (FromOrchestra, error) {
	for {
		select {
		case msg := <-e.recMsgChan:
			return FromOrchestra{msg: fmt.Sprintf("%v", msg)}, nil
		}
	}
	return FromOrchestra{}, nil
}

func NewExampleSubsystemContext(msgChan chan interface{}, recMsgChan chan interface{}) SubsystemContext {
	return &ExampleSubsystemContext{
		msgChan:    msgChan,
		recMsgChan: recMsgChan,
	}
}

func TestOrchestra(t *testing.T) {
	toOrchMsgChan := make(chan interface{})
	fromOrchMsgChan := make(chan interface{})
	subsystemContext := NewExampleSubsystemContext(toOrchMsgChan, fromOrchMsgChan)
	orchestra := NewOrchestra(toOrchMsgChan, fromOrchMsgChan)

	subSystemA := NewExampleSubsystem("Subsystem A")
	orchestra.AddSubsystem(subSystemA)
	orchestra.AddSubsystem(NewExampleSubsystem("Subsystem B"))

	orchestra.Start(subsystemContext)

	time.Sleep(5 * time.Second)
	fmt.Println("Stopping orchestra...")
	orchestra.Stop()

	fmt.Println("Orchestra stopped.")
}
