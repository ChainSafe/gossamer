// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package proof

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newRecorder(t *testing.T) {
	t.Parallel()

	expected := &recorder{}

	recorder := newRecorder()

	assert.Equal(t, expected, recorder)
}

func Test_recorder_record(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		recorder         *recorder
		hash             []byte
		rawData          []byte
		expectedRecorder *recorder
	}{
		"nil data": {
			recorder: &recorder{},
			expectedRecorder: &recorder{
				nodes: []visitedNode{
					{},
				},
			},
		},
		"insert in empty recorder": {
			recorder: &recorder{},
			hash:     []byte{1, 2},
			rawData:  []byte{3, 4},
			expectedRecorder: &recorder{
				nodes: []visitedNode{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
				},
			},
		},
		"insert in non-empty recorder": {
			recorder: &recorder{
				nodes: []visitedNode{
					{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
				},
			},
			hash:    []byte{1, 2},
			rawData: []byte{3, 4},
			expectedRecorder: &recorder{
				nodes: []visitedNode{
					{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testCase.recorder.record(testCase.hash, testCase.rawData)

			assert.Equal(t, testCase.expectedRecorder, testCase.recorder)
		})
	}
}

func Test_recorder_getNodes(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		recorder *recorder
		nodes    []visitedNode
	}{
		"no node": {
			recorder: &recorder{},
		},
		"get single node from recorder": {
			recorder: &recorder{
				nodes: []visitedNode{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
				},
			},
			nodes: []visitedNode{{Hash: []byte{1, 2}, RawData: []byte{3, 4}}},
		},
		"get node from multiple nodes in recorder": {
			recorder: &recorder{
				nodes: []visitedNode{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
					{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
					{Hash: []byte{9, 6}, RawData: []byte{7, 8}},
				},
			},
			nodes: []visitedNode{
				{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
				{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
				{Hash: []byte{9, 6}, RawData: []byte{7, 8}},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			nodes := testCase.recorder.getNodes()

			assert.Equal(t, testCase.nodes, nodes)
		})
	}
}
