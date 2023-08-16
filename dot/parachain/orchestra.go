package parachain

import (
	"context"
	"fmt"
	"sync"
)

type FromOrchestra struct{}

type SubsystemSender interface{}

type Subsystem interface {
	Start(ctx context.Context, out chan<- string)
	Stop()
}

// SubsystemContext defines the methods that should be implemented by a SubsystemContext
type SubsystemContext interface {
	TryRecv() (FromOrchestra, error)
	Recv() (FromOrchestra, error)
	Spawn(name string, f func(ctx context.Context)) error
	SpawnBlocking(name string, f func(context.Context)) error
	SendMessage(msg interface{})
	SendMessages(msgs []interface{})
	SendUnboundedMessage(msg interface{})
	Sender() SubsystemSender
}

type SpawnedSubsystem struct {
	Name   string
	Future func(ctx context.Context)
}

type Orchestra struct {
	subsystems []Subsystem
	wg         sync.WaitGroup
	msgChan    chan string
}

func NewOrchestra() *Orchestra {
	return &Orchestra{}
}

func (o *Orchestra) AddSubsystem(subsystem Subsystem) {
	o.subsystems = append(o.subsystems, subsystem)
}

func (o *Orchestra) Start(ctx context.Context) {
	o.msgChan = make(chan string)
	for _, subsystem := range o.subsystems {
		o.wg.Add(1)
		go func(subsystem Subsystem) {
			defer o.wg.Done()
			subsystem.Start(ctx, o.msgChan)
		}(subsystem)
	}
	go func() {
		fmt.Printf("In chan check")
		for {
			select {
			case msg := <-o.msgChan:
				fmt.Printf("Msg %v\n", msg)
			}
		}
	}()
}

func (o *Orchestra) Stop() {
	for _, subsystem := range o.subsystems {
		subsystem.Stop()
	}
	o.wg.Wait()
}
