package parachain

import (
	"context"
	"fmt"
	"testing"
	"time"
)

type ValidationSubsystem struct{}

func (v ValidationSubsystem) Start(ctx SubsystemContext) SpawnedSubsystem {
	return SpawnedSubsystem{
		Name: "validation-subsystem",
		Future: func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
				}
			}
		},
	}
}

type DummyOverseerBuilder struct{}

func (ob *DummyOverseerBuilder) Build() (Overseer, error) {
	return Overseer{}, nil
}

func NewDummyOverseerBuilder() OverseerBuilder {
	return &DummyOverseerBuilder{}
}

func TestOverseerWorks(t *testing.T) {
	//ctx := context.Background()

	//alwaysSupportsParachains := AlwaysSupportsParachains{}

	//spawner := NewTaskExecutor()
	//overseerBuilder := NewDummyOverseerBuilder(spawner, alwaysSupportsParachains, nil).
	//	WithCandidateValidation(func() ValidationSubsystem {
	//		return ValidationSubsystem{}
	//	})

	overseerBuilder := NewDummyOverseerBuilder()

	overseer, _ := overseerBuilder.Build()
	//overseerCtx, overseerCancel := context.WithCancel(ctx)

	timer := time.NewTimer(time.Millisecond * 50)
	defer timer.Stop()

	go func() {
		overseer.Run()
	}()
	//go func() {
	//	select {
	//	case <-overseer.Run(overseerCtx):
	//	case <-timer.C:
	//	}
	//	overseerCancel()
	//}()

	<-timer.C
	fmt.Println("Timer expired")
}
