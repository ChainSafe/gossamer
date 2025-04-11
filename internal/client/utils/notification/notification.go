package notification

import "github.com/ChainSafe/gossamer/internal/client/utils/pubsub"

// / The receiving half of the notifications channel(s).
// #[derive(Debug)]
//
//	pub struct NotificationReceiver<Payload> {
//		receiver: Receiver<Payload, Registry>,
//	}
type NotificationReceiver[Payload any] struct {
	pubsub.Receiver[func() (Payload, error), *registry[Payload]]
}

// / The sending half of the notifications channel(s).
//
//	pub struct NotificationSender<Payload> {
//		hub: Hub<Payload, Registry>,
//	}
type NotificationSender[Payload any] struct {
	pubsub.Hub[struct{}, func() (Payload, error), Payload, *registry[Payload]]
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

//	impl<Payload> NotificationSender<Payload> {
//		/// Send out a notification to all subscribers that a new payload is available for a
//		/// block.
//		pub fn notify<Error>(
//			&self,
//			payload: impl FnOnce() -> Result<Payload, Error>,
//		) -> Result<(), Error>
//		where
//			Payload: Clone,
//		{
func (ns *NotificationSender[Payload]) Notify(makePayload func() (Payload, error)) error {
	//		self.hub.send(payload)
	return ns.Hub.Send(makePayload)
}

// impl<Payload> Clone for NotificationSender<Payload> {
// 	fn clone(&self) -> Self {
// 		Self { hub: self.hub.clone() }
// 	}
// }
