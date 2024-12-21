package pubsub

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("client-utils", "pubsub"))

//! Provides means to implement a typical Pub/Sub mechanism.
//!
//! This module provides a type [`Hub`] which can be used both to subscribe,
//! and to send the broadcast messages.
//!
//! The [`Hub`] type is parametrized by two other types:
//! - `Message` — the type of a message that shall be delivered to the subscribers;
//! - `Registry` — implementation of the subscription/dispatch logic.
//!
//! A Registry is implemented by defining the following traits:
//! - [`Subscribe<K>`];
//! - [`Dispatch<M>`];
//! - [`Unsubscribe`].
//!
//! As a result of subscription `Hub::subscribe` method returns an instance of
//! [`Receiver<Message,Registry>`]. That can be used as a [`Stream`] to receive the messages.
//! Upon drop the [`Receiver<Message, Registry>`] shall unregister itself from the `Hub`.

// /// Unsubscribe: unregisters a previously created subscription.
// pub trait Unsubscribe {
type Unsubscribe interface {
	/// Remove all registrations of the subscriber with ID `subs_id`.
	// fn unsubscribe(&mut self, subs_id: SeqID);
	Unsubscribe(subsID uint64)
}

// / Subscribe using a key of type `K`
// pub trait Subscribe<K> {
type Subscribe[K any] interface {
	/// Register subscriber with the ID `subs_id` as having interest to the key `K`.
	// fn subscribe(&mut self, subs_key: K, subs_id: SeqID);
	Subscribe(subKey K, subsID uint64)
}

// / Dispatch a message of type `M`.
// pub trait Dispatch<M> {
type Dispatch[M, Item any] interface {
	// 	/// The type of the that shall be sent through the channel as a result of such dispatch.
	// 	type Item;
	// 	/// The type returned by the `dispatch`-method.
	// 	type Ret;

	/// Dispatch the message of type `M`.
	///
	/// The implementation is given an instance of `M` and is supposed to invoke `dispatch` for
	/// each matching subscriber, with an argument of type `Self::Item` matching that subscriber.
	///
	/// Note that this does not have to be of the same type with the item that will be sent through
	/// to the subscribers. The subscribers will receive a message of type `Self::Item`.
	// fn dispatch<F>(&mut self, message: M, dispatch: F) -> Self::Ret
	// where
	//
	//	F: FnMut(&SeqID, Self::Item);
	Dispatch(message M, dispatch func(seqID uint64, item Item))
}

type Registry[K any, M, Item any] interface {
	Subscribe[K]
	Unsubscribe
	Dispatch[M, Item]
}

// / A subscription hub.
// /
// / Does the subscription and dispatch.
// / The exact subscription and routing behaviour is to be implemented by the Registry (of type `R`).
// / The Hub under the hood uses the channel defined in `crate::mpsc` module.
// pub struct Hub<M, R> {
type Hub[K, M, Item any, R Registry[K, M, Item]] struct {
	// 	tracing_key: &'static str,
	tracingKey string
	//	shared: Arc<ReentrantMutex<RefCell<Shared<M, R>>>>,
	shared[Item, R]
}

// Create a new instance of Hub over the initialized Registry.
func NewHub[K any, M, Item any, R Registry[K, M, Item]](tracingKey string, registry R) *Hub[K, M, Item, R] {
	return &Hub[K, M, Item, R]{
		tracingKey: tracingKey,
		shared: shared[Item, R]{
			channels: make(map[uint64]chan<- Item),
			registry: registry,
		},
	}
}

// / Subscribe to this Hub using the `subs_key: K`.
// /
// / A subscription with a key `K` is possible if the Registry implements `Subscribe<K>`.
func (h *Hub[K, M, Item, R]) Subscribe(subsKey K) *Receiver[Item, R] {
	h.shared.Lock()
	defer h.shared.Unlock()

	subsID := h.shared.idSequence
	h.shared.idSequence++

	// The order (registry.subscribe then sinks.insert) is important here:
	// assuming that `Subscribe<K>::subscribe` can panic, it is better to at least
	// have the sink disposed.
	h.shared.registry.Subscribe(subsKey, subsID)

	_, ok := h.shared.channels[subsID]
	if ok {
		panic("Used IDSequence to create another ID. Should be unique until u64 is overflowed. Should be unique.")
	}
	ch := make(chan Item)
	h.shared.channels[subsID] = ch

	return &Receiver[Item, R]{
		channel: ch,
		shared:  &h.shared,
		subsID:  subsID,
	}
}

func (h *Hub[K, M, Item, R]) Send(trigger M) {
	h.shared.Lock()
	defer h.shared.Unlock()

	h.shared.registry.Dispatch(trigger, func(subsID uint64, item Item) {
		_, ok := h.shared.channels[subsID]
		if !ok {
			logger.Warnf("No Sink for SubsID = %d", subsID)
		}
		h.shared.channels[subsID] <- item
	})
}

func (h *Hub[K, M, Ret, R]) Shutdown() {
	h.shared.Lock()
	defer h.shared.Unlock()

	for _, ch := range h.shared.channels {
		close(ch)
	}
	h.shared.channels = nil
}

func (h *Hub[K, M, Ret, R]) Registry() R {
	return h.shared.registry
}

// / The receiving side of the subscription.
// /
// / The messages are delivered as items of a [`Stream`].
// / Upon drop this receiver unsubscribes itself from the [`Hub<M, R>`].
// pub struct Receiver<M, R>
type Receiver[M any, Registry Unsubscribe] struct {
	// rx: TracingUnboundedReceiver<M>,
	channel <-chan M
	// shared: Weak<ReentrantMutex<RefCell<Shared<M, R>>>>,
	shared *shared[M, Registry]
	// subs_id: SeqID,
	subsID uint64
}

func (r *Receiver[M, Registry]) Chan() <-chan M {
	return r.channel
}

func (r *Receiver[M, Registry]) Drop() {
	r.shared.Unsubscribe(r.subsID)
}

// #[derive(Debug)]
// struct Shared<M, R> {
type shared[M any, Registry Unsubscribe] struct {
	// 	id_sequence: crate::id_sequence::IDSequence,
	idSequence uint64
	// registry: R,
	registry Registry
	// sinks: HashMap<SeqID, TracingUnboundedSender<M>>,
	channels map[uint64]chan<- M

	sync.Mutex
}

func (s *shared[M, Registry]) Unsubscribe(subsID uint64) {
	s.Lock()
	defer s.Unlock()
	// The order (sinks.remove then registry.unsubscribe) is important here:
	// assuming that `Unsubscribe::unsubscribe` can panic, it is better to at least
	// have the sink disposed.
	_, ok := s.channels[subsID]
	if !ok {
		panic(fmt.Sprintf("invalid subsID: %d", subsID))
	}
	close(s.channels[subsID])
	delete(s.channels, subsID)
	s.registry.Unsubscribe(subsID)
}
