package parachain

type Spawner interface {
	Spawn(name string, f func())
	SpawnBlocking(name string, f func())
}

type TestSubsystemContext struct {
	tx    TestSubsystemSender
	rx    chan Message
	spawn Spawner
}

type TestSubsystemSender struct {
	tx chan<- Message
}

type Message interface{} // TODO: replace with actual message type

type AllMessages struct{} // TODO: replace with actuall AllMessages type

type TestSubsystemContextHandle struct {
	tx chan<- Message
	rx <-chan Message
}

func (s *TestSubsystemContextHandle) Send(fromOverseer Message) {
	s.tx <- fromOverseer
}

func makeSubsystemContext(spawner Spawner) (*TestSubsystemContext, *TestSubsystemContextHandle) {
	tx := make(chan Message)
	rx := make(chan Message)
	txAllMessages := make(chan Message)

	context := &TestSubsystemContext{
		tx:    TestSubsystemSender{tx: txAllMessages},
		rx:    rx,
		spawn: spawner,
	}

	handle := &TestSubsystemContextHandle{
		tx: tx,
		rx: rx,
	}

	return context, handle
}
