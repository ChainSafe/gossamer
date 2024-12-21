package api

import (
	"maps"

	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/storage"
)

var logger = log.NewFromGlobal(log.AddContext("client", "api"))

type ChildFilterKeys struct {
	Key  storage.StorageKey
	Keys []storage.StorageKey // can be nil
}

// / A command to subscribe with the specified filters.
// /
// / Used by the implementation of [`Subscribe<Op>`] trait for [`Registry].
// pub(super) struct SubscribeOp<'a> {
type SubscribeOp struct {
	// 	pub filter_keys: Option<&'a [StorageKey]>,
	FilterKeys []storage.StorageKey // can be nil
	// pub filter_child_keys: Option<&'a [(StorageKey, Option<Vec<StorageKey>>)]>,
	FilterChildKeys []ChildFilterKeys
}

type subscriberSink struct {
	subsID       uint64
	keys         Keys
	childKeys    ChildKeys
	wasTriggered bool
}

type registry[H runtime.Hash] struct {
	wildcardListeners map[uint64]any
	listeners         map[string]map[uint64]any
	childListeners    map[string]struct {
		cListeners map[string]map[uint64]any
		cWildcards map[uint64]any
	}
	sinks map[uint64]subscriberSink
}

func newRegistry[H runtime.Hash]() *registry[H] {
	return &registry[H]{
		wildcardListeners: make(map[uint64]any),
		listeners:         make(map[string]map[uint64]any),
		childListeners: make(map[string]struct {
			cListeners map[string]map[uint64]any
			cWildcards map[uint64]any
		}),
		sinks: make(map[uint64]subscriberSink),
	}
}

func (r *registry[H]) Subscribe(subsOp SubscribeOp, subsID uint64) {
	keys := r.listenFrom(subsID, subsOp.FilterKeys, r.listeners, r.wildcardListeners)

	var childKeys map[string]map[string]any = nil
	if subsOp.FilterChildKeys != nil {
		for _, fck := range subsOp.FilterChildKeys {
			cKey := fck.Key
			oKeys := fck.Keys

			_, ok := r.childListeners[string(cKey)]
			if !ok {
				r.childListeners[string(cKey)] = struct {
					cListeners map[string]map[uint64]any
					cWildcards map[uint64]any
				}{
					cListeners: make(map[string]map[uint64]any),
					cWildcards: make(map[uint64]any),
				}
			}
			if childKeys == nil {
				childKeys = make(map[string]map[string]any)
			}
			childKeys[string(cKey)] = r.listenFrom(subsID, oKeys, r.childListeners[string(cKey)].cListeners, r.childListeners[string(cKey)].cWildcards)
		}
	}

	// if let Some(m) = self.metrics.as_ref() {
	// 	m.with_label_values(&["added"]).inc();
	// }

	_, ok := r.sinks[subsID]
	if ok {
		logger.Warnf("subscribe has been passed a non-unique subsID")
	}
	r.sinks[subsID] = subscriberSink{
		subsID:       subsID,
		keys:         keys,
		childKeys:    childKeys,
		wasTriggered: false,
	}
}

func (r *registry[H]) Unsubscribe(subsID uint64) {
	r.removeSubscriber(subsID)
}

func (r *registry[H]) removeSubscriber(subscriber uint64) *struct {
	Keys
	ChildKeys
} {
	sink, ok := r.sinks[subscriber]
	if !ok {
		return nil
	}
	delete(r.sinks, subscriber)

	r.removeSubscriberFrom(subscriber, sink.keys, r.listeners, r.wildcardListeners)
	if sink.childKeys != nil {
		for cKey, filters := range sink.childKeys {
			_, ok := r.childListeners[cKey]
			if ok {
				r.removeSubscriberFrom(subscriber, filters, r.childListeners[cKey].cListeners, r.childListeners[cKey].cWildcards)
			}

			if len(r.childListeners[cKey].cListeners) == 0 && len(r.childListeners[cKey].cWildcards) == 0 {
				delete(r.childListeners, cKey)
			}

		}
	}

	// if let Some(m) = self.metrics.as_ref() {
	// 	m.with_label_values(&["removed"]).inc();
	// }

	return &struct {
		Keys
		ChildKeys
	}{
		Keys:      sink.keys,
		ChildKeys: sink.childKeys,
	}
}

func (r *registry[H]) removeSubscriberFrom(subscriber uint64, filters Keys, listeners map[string]map[uint64]any, wildcards map[uint64]any) {
	if filters == nil {
		delete(wildcards, subscriber)
	} else {
		for key := range filters {
			var removeKey bool
			_, ok := listeners[key]
			if ok {
				delete(listeners[key], subscriber)
				removeKey = len(listeners[key]) == 0
			} else {
				removeKey = false
			}

			if removeKey {
				delete(listeners, key)
			}
		}
	}
}

func (r *registry[H]) listenFrom(
	currentID uint64,
	filterKeys []storage.StorageKey,
	listeners map[string]map[uint64]any,
	wildcards map[uint64]any,
) Keys {
	if filterKeys == nil {
		wildcards[currentID] = nil
		return nil
	}
	keys := make(Keys)
	for _, key := range filterKeys {
		_, ok := listeners[string(key)]
		if !ok {
			listeners[string(key)] = make(map[uint64]any)
		}
		listeners[string(key)][currentID] = nil
		keys[string(key)] = nil
	}
	return keys
}

type Message[H any] struct {
	Hash           H
	ChangeSet      []Change
	ChildChangeSet []ChildChange
}

func (r *registry[H]) Dispatch(message Message[H], dispatch func(uint64, StorageNotification[H])) {
	r.trigger(message.Hash, message.ChangeSet, message.ChildChangeSet, dispatch)
}

func (r *registry[H]) trigger(hash H, changeset []Change, childChangeSet []ChildChange, dispatch func(uint64, StorageNotification[H])) {
	hasWildcard := len(r.wildcardListeners) != 0

	// early exit if no listeners
	if !hasWildcard && len(r.listeners) == 0 && len(r.childListeners) == 0 {
		return
	}

	subscribers := maps.Clone(r.wildcardListeners)
	var changes []Change
	var childChanges []ChildChange

	// collect subscribers and changes
	for _, change := range changeset {
		listeners, ok := r.listeners[string(change.Key)]
		if ok {
			for listener := range listeners {
				subscribers[listener] = nil
			}
		}

		if hasWildcard || len(listeners) > 0 {
			changes = append(changes, Change{
				Key:   change.Key,
				Value: change.Value,
			})
		}
	}
	for _, childChange := range childChangeSet {
		childListener, ok := r.childListeners[string(childChange.StorageKey)]
		if ok {
			var changes []Change
			for _, change := range childChange.ChangeSet {
				listeners, ok := childListener.cListeners[string(change.Key)]

				if ok {
					for listener := range listeners {
						subscribers[listener] = nil
					}
				}

				for listener := range childListener.cWildcards {
					subscribers[listener] = nil
				}

				if len(childListener.cWildcards) > 0 || len(listeners) > 0 {
					changes = append(changes, Change{
						Key:   change.Key,
						Value: change.Value,
					})
				}
			}
			if len(changes) > 0 {
				childChanges = append(childChanges, ChildChange{
					StorageKey: childChange.StorageKey,
					ChangeSet:  changes,
				})
			}
		}
	}

	// Don't send empty notifications
	if len(changes) == 0 && len(childChanges) == 0 {
		return
	}

	// Trigger the events
	for subsID, sink := range r.sinks {
		if _, ok := subscribers[subsID]; ok {
			sink.wasTriggered = true
			r.sinks[subsID] = sink

			var (
				filteredChanges      []Change
				filteredChildChanges []ChildChange
			)

			if sink.keys != nil {
				for _, change := range changes {
					_, ok := sink.keys[string(change.Key)]
					if ok {
						filteredChanges = append(filteredChanges, change)
					}
				}
			} else {
				filteredChanges = changes
			}

			if sink.childKeys != nil {
				for _, childChange := range childChanges {
					filter, ok := sink.childKeys[string(childChange.StorageKey)]
					if ok {
						filteredChildChange := ChildChange{
							StorageKey: childChange.StorageKey,
							ChangeSet:  nil,
						}
						for _, change := range childChange.ChangeSet {
							if filter == nil {
								filteredChildChange.ChangeSet = append(filteredChildChange.ChangeSet, change)
							} else {
								_, ok := filter[string(change.Key)]
								if ok {
									filteredChildChange.ChangeSet = append(filteredChildChange.ChangeSet, change)
								}
							}
						}
						filteredChildChanges = append(filteredChildChanges, filteredChildChange)
					}
				}
			}

			storageChangeSet := StorageChangeSet{
				Changes:      filteredChanges,
				ChildChanges: filteredChildChanges,
				Filter:       sink.keys,
				ChildFilters: sink.childKeys,
			}

			notification := StorageNotification[H]{
				Block:            hash,
				StorageChangeSet: storageChangeSet,
			}

			dispatch(subsID, notification)
		}
	}
}
