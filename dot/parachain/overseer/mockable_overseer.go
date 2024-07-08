package overseer

import (
	"context"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

type MockableOverseer struct {
	// TODO Is it okay to have a pointer to testing.T here? Figure out!!
	t      *testing.T
	ctx    context.Context
	cancel context.CancelFunc

	SubsystemsToOverseer chan any
	overseerToSubsystem  chan any
	subSystem            parachaintypes.Subsystem

	// this is going to be limited to only one testcase
	expectedMessagesWithAction map[any]func(msg any)
}

func NewMockableOverseer(t *testing.T) *MockableOverseer {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	return &MockableOverseer{
		t:                    t,
		ctx:                  ctx,
		cancel:               cancel,
		SubsystemsToOverseer: make(chan any),

		expectedMessagesWithAction: make(map[any]func(msg any)),
	}
}
func (m *MockableOverseer) GetSubsystemToOverseerChannel() chan any {
	return m.SubsystemsToOverseer
}
func (m *MockableOverseer) RegisterSubsystem(subsystem parachaintypes.Subsystem) chan any {
	OverseerToSubSystem := make(chan any)
	m.overseerToSubsystem = OverseerToSubSystem
	m.subSystem = subsystem
	return OverseerToSubSystem
}
func (m *MockableOverseer) Start() error {
	go m.processMessages(m.t)

	return nil
}
func (m *MockableOverseer) Stop() {
	m.cancel()

}
func (m *MockableOverseer) ReceiveMessage(msg any) {
	m.overseerToSubsystem <- msg
}

type InputOutput struct {
	InputMessage  any
	OutputMessage any
}

func (m *MockableOverseer) ExpectMessageWithAction(msg any, fn func(msg any)) {
	m.expectedMessagesWithAction[msg] = fn
}

//	func test(msg any) {
//		newMessage := msg.(parachaintypes.ProspectiveParachainsMessageIntroduceCandidate)
//		newMessage.Ch <- errors.New("error")
//	}
func (m *MockableOverseer) processMessages(t *testing.T) {
	for {
		select {
		case msg := <-m.SubsystemsToOverseer:
			if msg == nil {
				continue
			}
			action, ok := m.expectedMessagesWithAction[msg]
			if !ok {
				t.Errorf("unexpected message: %v", msg)
				continue
			}

			action(msg)
		case <-m.ctx.Done():
			if err := m.ctx.Err(); err != nil {
				logger.Errorf("ctx error: %v\n", err)
			}
			return
		}
	}
}
