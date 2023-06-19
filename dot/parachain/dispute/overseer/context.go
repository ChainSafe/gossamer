// TODO: This is just a temporary file to complete the participation module. The type definitions here are not complete.
// We need to remove this file once we have implemented the overseer.

package overseer

type Sender interface {
	SendMessage(msg any) error
	Feed(msg any) error
}

type Context struct {
	Sender Sender
}
