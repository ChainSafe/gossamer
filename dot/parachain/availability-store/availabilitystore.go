package availability_store

import (
	"context"
	"fmt"
)

func (av *AvailabilityStore) Run(ctx context.Context, OverseerToSubsystem chan any, SubsystemToOverseer chan any) error {
	for {
		select {
		case msg, ok := <-av.OverseerToSubSystem:
			if !ok {
				return nil
			}
			err := av.processMessage(msg)
			if err != nil {
				return err
			}
		}
	}
}

func (av *AvailabilityStore) processMessage(msg interface{}) error {
	fmt.Printf("AvailabilityStore: Got message %v\n", msg)
	return nil
}

type AvailabilityStore struct {
	SubSystemToOverseer chan<- any
	OverseerToSubSystem <-chan any
}
