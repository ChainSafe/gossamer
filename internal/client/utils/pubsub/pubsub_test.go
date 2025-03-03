// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package pubsub_test

import (
	"testing"

	"github.com/ChainSafe/gossamer/internal/client/utils/pubsub"
	"github.com/stretchr/testify/require"
)

type Message uint64
type TestHub struct {
	*pubsub.Hub[SubsKey, Message, Message, *Registry[Message]]
}
type TestReceiver = pubsub.Receiver[Message, *Registry[Message]]

type Registry[M any] struct {
	subscribers map[uint64]SubsKey
}

func (r *Registry[M]) Subscribe(subsKey SubsKey, subsID uint64) {
	r.subscribers[subsID] = subsKey
}

func (r *Registry[M]) Unsubscribe(subsID uint64) {
	delete(r.subscribers, subsID)
}

func (r *Registry[M]) Dispatch(message M, dispatch func(uint64, M)) {
	for id, subsKey := range r.subscribers {
		_ = subsKey
		dispatch(id, message)
	}
}

type SubsKey struct {
	// receiver *TestReceiver
}

func NewTestHub() *TestHub {
	return &TestHub{
		pubsub.NewHub("a_tracing_key", &Registry[Message]{subscribers: map[uint64]SubsKey{}}),
	}
}

func (th *TestHub) SubsCount() int {
	r := th.Hub.Registry()
	return len(r.subscribers)
}

func Test_Hub(t *testing.T) {
	t.Run("receives_relevant_messages_and_chan_closes_on_hub_shutdown", func(t *testing.T) {
		hub := NewTestHub()
		require.Equal(t, 0, hub.SubsCount())

		// No subscribers yet. That message is not supposed to get to anyone.
		hub.Send(0)

		rx01 := hub.Subscribe(SubsKey{})
		require.Equal(t, 1, hub.SubsCount())

		// That message is sent after subscription. Should be delivered into rx_01.
		done := make(chan Message)
		go func() {
			m := <-rx01.Chan()
			require.Equal(t, Message(1), m)
			close(done)
		}()
		hub.Send(1)
		<-done

		// Hub is disposed, so rx01 should be closed.
		hub.Shutdown()

		done = make(chan Message)
		go func() {
			for range rx01.Chan() {
			}
			close(done)
		}()
		<-done
	})

	t.Run("subs_count_is_modified_on_rx_drop", func(t *testing.T) {
		hub := NewTestHub()
		require.Equal(t, 0, hub.SubsCount())

		rx01 := hub.Subscribe(SubsKey{})
		require.Equal(t, 1, hub.SubsCount())
		rx02 := hub.Subscribe(SubsKey{})
		require.Equal(t, 2, hub.SubsCount())

		rx01.Drop()
		require.Equal(t, 1, hub.SubsCount())
		rx02.Drop()
		require.Equal(t, 0, hub.SubsCount())
	})

	t.Run("positive_subs_count_is_correct_upon_drop_of_rxs_on_cloned_hubs", func(t *testing.T) {
		hub := NewTestHub()
		hub2 := hub
		require.Equal(t, 0, hub.SubsCount())
		require.Equal(t, 0, hub2.SubsCount())

		rx01 := hub2.Subscribe(SubsKey{})
		require.Equal(t, 1, hub.SubsCount())
		require.Equal(t, 1, hub2.SubsCount())

		rx02 := hub2.Subscribe(SubsKey{})
		require.Equal(t, 2, hub.SubsCount())
		require.Equal(t, 2, hub2.SubsCount())

		rx01.Drop()
		require.Equal(t, 1, hub.SubsCount())
		require.Equal(t, 1, hub2.SubsCount())

		rx02.Drop()
		require.Equal(t, 0, hub.SubsCount())
		require.Equal(t, 0, hub2.SubsCount())
	})
}
