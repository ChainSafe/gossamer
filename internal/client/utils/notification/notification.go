// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package notification

import "github.com/ChainSafe/gossamer/internal/client/utils/pubsub"

// / The receiving half of the notifications channel.
// /
// / The [`NotificationStream`] entity stores the [`Hub`] so it can be
// / used to add more subscriptions.
// #[derive(Clone)]
// pub struct NotificationStream<Payload, TK: TracingKeyStr> {
type NotificationStream[Payload any] struct {
	// hub: Hub<Payload, Registry>,
	hub *pubsub.Hub[struct{}, func() (Payload, error), Payload, *registry[Payload]]
	// _pd: std::marker::PhantomData<TK>,
}

// impl<Payload, TK: TracingKeyStr> NotificationStream<Payload, TK> {
// 	/// Creates a new pair of receiver and sender of `Payload` notifications.
// 	pub fn channel() -> (NotificationSender<Payload>, Self) {
// 		let hub = Hub::new(TK::TRACING_KEY);
// 		let sender = NotificationSender { hub: hub.clone() };
// 		let receiver = NotificationStream { hub, _pd: Default::default() };
// 		(sender, receiver)
// 	}

// 	/// Subscribe to a channel through which the generic payload can be received.
// 	pub fn subscribe(&self, queue_size_warning: usize) -> NotificationReceiver<Payload> {
// 		let receiver = self.hub.subscribe((), queue_size_warning);
// 		NotificationReceiver { receiver }
// 	}
// }

// NotificationSender is the sending half of the notifications channel(s).
type NotificationSender[Payload any] struct {
	hub *pubsub.Hub[struct{}, func() (Payload, error), Payload, *registry[Payload]]
}

// Notify sends out a notification to all subscribers that a new payload is available for a block.
func (ns *NotificationSender[Payload]) Notify(makePayload func() (Payload, error)) error {
	return ns.hub.Send(makePayload)
}
