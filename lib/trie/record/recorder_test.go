package record

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewRecorder(t *testing.T) {
	t.Parallel()

	expected := &Recorder{}

	recorder := NewRecorder()

	assert.Equal(t, expected, recorder)
}

func Test_Recorder_Record(t *testing.T) {
	testCases := map[string]struct {
		recorder         *Recorder
		hash             []byte
		rawData          []byte
		expectedRecorder *Recorder
	}{
		"nil data": {
			recorder: &Recorder{},
			expectedRecorder: &Recorder{
				nodes: []Node{
					{},
				},
			},
		},
		"insert in empty recorder": {
			recorder: &Recorder{},
			hash:     []byte{1, 2},
			rawData:  []byte{3, 4},
			expectedRecorder: &Recorder{
				nodes: []Node{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
				},
			},
		},
		"insert in non-empty recorder": {
			recorder: &Recorder{
				nodes: []Node{
					{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
				},
			},
			hash:    []byte{1, 2},
			rawData: []byte{3, 4},
			expectedRecorder: &Recorder{
				nodes: []Node{
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

			testCase.recorder.Record(testCase.hash, testCase.rawData)

			assert.Equal(t, testCase.expectedRecorder, testCase.recorder)
		})
	}
}

func Test_Recorder_GetNodes(t *testing.T) {
	testCases := map[string]struct {
		recorder *Recorder
		nodes    []Node
	}{
		"no node": {
			recorder: &Recorder{},
		},
		"get single node from recorder": {
			recorder: &Recorder{
				nodes: []Node{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
				},
			},
			nodes: []Node{{Hash: []byte{1, 2}, RawData: []byte{3, 4}}},
		},
		"get node from multiple nodes in recorder": {
			recorder: &Recorder{
				nodes: []Node{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
					{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
					{Hash: []byte{9, 6}, RawData: []byte{7, 8}},
				},
			},
			nodes: []Node{
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

			nodes := testCase.recorder.GetNodes()

			assert.Equal(t, testCase.nodes, nodes)
		})
	}
}
