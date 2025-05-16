// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package statementdistribution

import (
	"maps"
	"slices"

	"github.com/ChainSafe/gossamer/dot/parachain/grid"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

// groupSubView is our local view of a subset of the grid topology organized
// around a specific validator group.
//
// This tracks which authorities we expect to communicate with concerning
// candidates from the group. This includes both the authorities we are
// expected to send to as well as the authorities we expect to receive from.
//
// In the case that this group is the group that we are locally assigned to,
// the 'receiving' side will be empty.
type groupSubView struct {
	// validators we are 'sending' to.
	sending map[parachaintypes.ValidatorIndex]struct{}
	// validators we are 'receiving' from.
	receiving map[parachaintypes.ValidatorIndex]struct{}
}

// sessionTopologyView is our local view of the topology for a session, as
// it pertains to backed candidate distribution.
type sessionTopologyView struct {
	groupViews map[parachaintypes.GroupIndex]groupSubView
}

func newSessionTopology() *sessionTopologyView {
	return &sessionTopologyView{
		groupViews: make(map[parachaintypes.GroupIndex]groupSubView),
	}
}

// buildSessionTopologyView builds a view of the topology for the session.
// For groups that we are part of: we receive from nobody and send to our X/Y peers.
// For groups that we are not part of: we receive from any validator in the group we share a slice
// with and send to the corresponding X/Y slice in the other dimension.
//
// For any validators we don't share a slice with, we receive from the nodes
// which share a slice with them.
func buildSessionTopologyView(
	groups [][]parachaintypes.ValidatorIndex,
	topology *grid.SessionGridTopology,
	ourIndex *parachaintypes.ValidatorIndex,
) (*sessionTopologyView, error) {
	view := newSessionTopology()

	if ourIndex == nil {
		return view, nil
	}

	ourNeighbours, err := topology.ComputeGridNeighboursFor(*ourIndex)
	if err != nil {
		return nil, err
	}

	if ourNeighbours == nil {
		logger.Warnf("our index %d unrecognized in topology?", ourIndex)
		return view, nil
	}

	for i, group := range groups {
		subView := groupSubView{
			sending:   make(map[parachaintypes.ValidatorIndex]struct{}),
			receiving: make(map[parachaintypes.ValidatorIndex]struct{}),
		}

		if slices.Contains(group, *ourIndex) {
			maps.Copy(subView.sending, ourNeighbours.ValidatorIndicesRow)
			maps.Copy(subView.sending, ourNeighbours.ValidatorIndicesCol)

			// remove all other same-group validators from this set, they are
			// in the cluster.
			// TODO [now]: test this behavior. (from polkadot-sdk)
			for _, validatorIndex := range group {
				delete(subView.sending, validatorIndex)
			}
		} else {
			rowNeighboursWithoutGroup := make(map[parachaintypes.ValidatorIndex]struct{})
			for validatorIndex, _ := range ourNeighbours.ValidatorIndicesRow {
				if !slices.Contains(group, validatorIndex) {
					rowNeighboursWithoutGroup[validatorIndex] = struct{}{}
				}
			}

			colNeighboursWithoutGroup := make(map[parachaintypes.ValidatorIndex]struct{})
			for validatorIndex, _ := range ourNeighbours.ValidatorIndicesCol {
				if !slices.Contains(group, validatorIndex) {
					colNeighboursWithoutGroup[validatorIndex] = struct{}{}
				}
			}

			for _, validatorIndex := range group {
				// If the validator shares a slice with us, we expect to
				// receive from them and send to our neighbors in the other
				// dimension.

				if _, ok := ourNeighbours.ValidatorIndicesRow[validatorIndex]; ok {
					subView.receiving[validatorIndex] = struct{}{}
					maps.Copy(subView.sending, colNeighboursWithoutGroup)
					continue
				}

				if _, ok := ourNeighbours.ValidatorIndicesCol[validatorIndex]; ok {
					subView.receiving[validatorIndex] = struct{}{}
					maps.Copy(subView.sending, rowNeighboursWithoutGroup)
					continue
				}

				// If they don't share a slice with us, we don't send to anybody
				// but receive from any peers sharing a dimension with both of us
				theirNeighbours, err := topology.ComputeGridNeighboursFor(validatorIndex)
				if err != nil {
					return nil, err
				}

				if theirNeighbours == nil {
					logger.Warnf("validator index %d unrecognized in topology?", validatorIndex)
					continue
				}

				// their X, our Y
				for potentialLink, _ := range theirNeighbours.ValidatorIndicesRow {
					if _, ok := ourNeighbours.ValidatorIndicesCol[potentialLink]; ok {
						subView.receiving[potentialLink] = struct{}{}
						break // one max
					}
				}

				// their Y, our X
				for potentialLink, _ := range theirNeighbours.ValidatorIndicesCol {
					if _, ok := ourNeighbours.ValidatorIndicesRow[potentialLink]; ok {
						subView.receiving[potentialLink] = struct{}{}
						break // one max
					}
				}
			}
		}

		view.groupViews[parachaintypes.GroupIndex(i)] = subView
	}

	return view, nil
}
