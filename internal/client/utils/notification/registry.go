// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package notification

// The shared structure to keep track on subscribers.
type registry[Payload any] struct {
	subscribers map[uint64]struct{}
}

func (r *registry[Payload]) Subscribe(_subsKey struct{}, subsID uint64) {
	r.subscribers[subsID] = struct{}{}
}

func (r *registry[Payload]) Unsubscribe(subsID uint64) {
	delete(r.subscribers, subsID)
}

func (r *registry[Payload]) Dispatch(makePayload func() (Payload, error), dispatch func(uint64, Payload)) error {
	if len(r.subscribers) == 0 {
		return nil
	}

	payload, err := makePayload()
	if err != nil {
		return err
	}

	for subscriber := range r.subscribers {
		dispatch(subscriber, payload)
	}
	return nil
}
