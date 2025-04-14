package notification

import "github.com/ChainSafe/gossamer/internal/client/utils/pubsub"

// NotificationSender is the sending half of the notifications channel(s).
type NotificationSender[Payload any] struct {
	pubsub.Hub[struct{}, func() (Payload, error), Payload, *registry[Payload]]
}

// Notify sends out a notification to all subscribers that a new payload is available for a block.
func (ns *NotificationSender[Payload]) Notify(makePayload func() (Payload, error)) error {
	return ns.Hub.Send(makePayload)
}
