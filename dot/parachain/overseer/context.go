package overseer

type Context struct {
	Sender   Sender
	Receiver chan any
}

type Sender interface {
	SendMessage(msg any) error
	Feed(msg any) error
}
