package parachain

import (
	"context"
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

func (s *ExampleSubsystem) Start(ctx context.Context, out chan<- string) {
	fmt.Printf("Subsystem %s started\n", s.name)
	for {
		select {
		case <-s.done:
			fmt.Printf("Subsystem %s stopped\n", s.name)
			return
		case <-ctx.Done():
			fmt.Printf("Subsystem %s received cancel signal\n", s.name)
			return
		default:
			out <- fmt.Sprintf("Subsystem %s working...", s.name)
			time.Sleep(time.Second)
		}
	}
}

func (s *ExampleSubsystem) Stop() {
	close(s.done)
}

func TestOrchestra(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orchestra := NewOrchestra()
	orchestra.AddSubsystem(NewExampleSubsystem("Subsystem A"))
	orchestra.AddSubsystem(NewExampleSubsystem("Subsystem B"))

	orchestra.Start(ctx)

	time.Sleep(5 * time.Second)
	fmt.Println("Stopping orchestra...")
	orchestra.Stop()

	fmt.Println("Orchestra stopped.")
}
