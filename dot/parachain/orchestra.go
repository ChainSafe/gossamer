package parachain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type FromOrchestra struct {
	msg string
}

type SubsystemSender interface{}

type Subsystem interface {
	Start(subCtx SubsystemContext)
	Stop()
}

// SubsystemContext defines the methods that should be implemented by a SubsystemContext
type SubsystemContext interface {
	//TryRecv() (FromOrchestra, error)
	Recv() (FromOrchestra, error)
	//Spawn(name string, f func(ctx context.Context)) error
	//SpawnBlocking(name string, f func(context.Context)) error
	SendMessage(msg interface{})
	//SendMessages(msgs []interface{})
	//SendUnboundedMessage(msg interface{})
	//Sender() SubsystemSender
}

type SpawnedSubsystem struct {
	Name   string
	Future func(ctx context.Context)
}

type Orchestra struct {
	subsystems       []Subsystem
	wg               sync.WaitGroup
	recMsgChan       <-chan interface{}
	sendMsgChan      chan interface{}
	subsystemContext SubsystemContext
}

func NewOrchestra(recMsgChan <-chan interface{}, sendMsgChan chan interface{}) *Orchestra {
	return &Orchestra{
		recMsgChan:  recMsgChan,
		sendMsgChan: sendMsgChan,
	}
}

func (o *Orchestra) AddSubsystem(subsystem Subsystem) {
	o.subsystems = append(o.subsystems, subsystem)
}

func (o *Orchestra) Start(ctx SubsystemContext) {
	o.subsystemContext = ctx
	for _, subsystem := range o.subsystems {
		o.wg.Add(1)
		go func(subsystem Subsystem) {
			defer o.wg.Done()
			subsystem.Start(o.subsystemContext)
		}(subsystem)
	}
	go func() {
		for {
			select {
			case msg := <-o.recMsgChan:
				fmt.Printf("Orchesta received msg: %v\n", msg)
			}
		}
	}()
	go func() {
		for {
			time.Sleep(time.Millisecond * 1200)
			fmt.Printf("sending message\n")
			o.sendMsgChan <- fmt.Sprintf("Msg From Orchastra")
		}
	}()
}

func (o *Orchestra) Stop() {
	for _, subsystem := range o.subsystems {
		subsystem.Stop()
	}
	o.wg.Wait()
}
