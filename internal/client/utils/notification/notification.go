// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package notification

import "github.com/ChainSafe/gossamer/internal/client/utils/pubsub"

// The receiving half of the notifications channel.
//
// The [NotificationStream] entity stores the [pubsub.Hub] so it can be
// used to add more subscriptions.
type NotificationStream[Payload any] struct {
	hub *pubsub.Hub[struct{}, func() (Payload, error), Payload, *registry[Payload]] //nolint: unused
}

//	impl<Payload, TK: TracingKeyStr> NotificationStream<Payload, TK> {
//		/// Creates a new pair of receiver and sender of `Payload` notifications.
//		pub fn channel() -> (NotificationSender<Payload>, Self) {
//			let hub = Hub::new(TK::TRACING_KEY);
//			let sender = NotificationSender { hub: hub.clone() };
//			let receiver = NotificationStream { hub, _pd: Default::default() };
//			(sender, receiver)
//		}
func NewNotificationStream[Payload any]() (NotificationSender[Payload], NotificationStream[Payload]) {
	hub := pubsub.NewHub("", &registry[Payload]{})
	sender := NotificationSender[Payload]{hub: hub}
	receiver := NotificationStream[Payload]{hub: hub}
	return sender, receiver
}

// NotificationSender is the sending half of the notifications channel(s).
type NotificationSender[Payload any] struct {
	hub *pubsub.Hub[struct{}, func() (Payload, error), Payload, *registry[Payload]]
}

// Notify sends out a notification to all subscribers that a new payload is available for a block.
func (ns *NotificationSender[Payload]) Notify(makePayload func() (Payload, error)) error {
	return ns.hub.Send(makePayload)
}
