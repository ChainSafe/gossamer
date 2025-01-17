// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// Provides means to implement a typical Pub/Sub mechanism.
//
// This module provides a type [Hub] which can be used both to subscribe,
// and to send the broadcast messages.
//
// The [`Hub`] type is parametrized by two other types:
// - M — the type of a message that shall be delivered to the subscribers;
// - Registry — implementation of the subscription/dispatch logic.
//
// A Registry is implemented by defining the following traits:
// - [Subscribe]
// - [Dispatch]
// - [Unsubscribe]
//
// As a result of subscription [Hub.Subscribe] method returns an instance of
// [Receiver]. That can be used to retrieve a channel of messages.
// Upon [Receiver.Drop] the [Receiver] shall unregister itself from the [Hub].
package pubsub

import (
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("client-utils", "pubsub"))

// Unsubscribe unregisters a previously created subscription.
type Unsubscribe interface {
	// Remove all registrations of the subscriber with ID subsID.
	Unsubscribe(subsID uint64)
}

// Subscribe using a key of type K
type Subscribe[K any] interface {
	// Register subscriber with the ID `subs_id` as having interest to the key `K`.
	// fn subscribe(&mut self, subs_key: K, subs_id: SeqID);
	Subscribe(subKey K, subsID uint64)
}

// Dispatch a message of type M. The type Item will be sent through the channel as a result of such dispatch.
type Dispatch[M, Item any] interface {
	// Dispatch the message of type M.
	//
	// The implementation is given an instance of M and is supposed to invoke Dispatch for
	// each matching subscriber, with an argument of type Item matching that subscriber.
	//
	// Note that this does not have to be of the same type with the item that will be sent through
	// to the subscribers. The subscribers will receive a message of type Item.
	Dispatch(message M, dispatch func(seqID uint64, item Item))
}

type Registry[K any, M, Item any] interface {
	Subscribe[K]
	Unsubscribe
	Dispatch[M, Item]
}

// Hub is a subscription hub.
//
// Does the subscription and dispatch.
// The exact subscription and routing behaviour is to be implemented by the Registry (of type R).
type Hub[K, M, Item any, R Registry[K, M, Item]] struct {
	// 	tracing_key: &'static str,
	tracingKey string
	//	shared: Arc<ReentrantMutex<RefCell<Shared<M, R>>>>,
	shared[Item, R]
}

// NewHub creates a new instance of Hub over the initialised Registry.
func NewHub[K any, M, Item any, R Registry[K, M, Item]](tracingKey string, registry R) *Hub[K, M, Item, R] {
	return &Hub[K, M, Item, R]{
		tracingKey: tracingKey,
		shared: shared[Item, R]{
			channels: make(map[uint64]chan<- Item),
			registry: registry,
		},
	}
}

// Subscribe to this Hub using the subsKey.
//
// A subscription with a key K is possible if the Registry implements [Subscribe].
func (h *Hub[K, M, Item, R]) Subscribe(subsKey K) *Receiver[Item, R] {
	h.shared.Lock()
	defer h.shared.Unlock()

	subsID := h.shared.idSequence
	h.shared.idSequence++

	h.shared.registry.Subscribe(subsKey, subsID)

	_, ok := h.shared.channels[subsID]
	if ok {
		panic("Used subsID to create another ID. Should be unique until uint64 is overflowed.")
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

// Receiver is the receiving side of the subscription.
//
// The messages are delivered as items from a channel.
// Upon calling Drop this receiver unsubscribes itself from the [Hub].
type Receiver[M any, Registry Unsubscribe] struct {
	channel <-chan M
	shared  *shared[M, Registry]
	subsID  uint64
}

func (r *Receiver[M, Registry]) Chan() <-chan M {
	return r.channel
}

func (r *Receiver[M, Registry]) Drop() {
	r.shared.Unsubscribe(r.subsID)
}

type shared[M any, Registry Unsubscribe] struct {
	idSequence uint64
	registry   Registry
	channels   map[uint64]chan<- M
	sync.Mutex
}

func (s *shared[M, Registry]) Unsubscribe(subsID uint64) {
	s.Lock()
	defer s.Unlock()
	_, ok := s.channels[subsID]
	if !ok {
		panic(fmt.Sprintf("invalid subsID: %d", subsID))
	}
	close(s.channels[subsID])
	delete(s.channels, subsID)
	s.registry.Unsubscribe(subsID)
}
