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

func Test_Recorder_Next(t *testing.T) {
	testCases := map[string]struct {
		recorder         *Recorder
		node             Node
		err              error
		expectedRecorder *Recorder
	}{
		"no node": {
			recorder:         &Recorder{},
			err:              ErrNoNextNode,
			expectedRecorder: &Recorder{},
		},
		"get single node from recorder": {
			recorder: &Recorder{
				nodes: []Node{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
				},
			},
			node: Node{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
			expectedRecorder: &Recorder{
				nodes: []Node{},
			},
		},
		"get node from multiple nodes in recorder": {
			recorder: &Recorder{
				nodes: []Node{
					{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
					{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
					{Hash: []byte{9, 6}, RawData: []byte{7, 8}},
				},
			},
			node: Node{Hash: []byte{1, 2}, RawData: []byte{3, 4}},
			expectedRecorder: &Recorder{
				nodes: []Node{
					{Hash: []byte{5, 6}, RawData: []byte{7, 8}},
					{Hash: []byte{9, 6}, RawData: []byte{7, 8}},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			node, err := testCase.recorder.Next()

			assert.ErrorIs(t, err, testCase.err)
			if testCase.err != nil {
				assert.EqualError(t, err, testCase.err.Error())
			}
			assert.Equal(t, testCase.node, node)
			assert.Equal(t, testCase.expectedRecorder, testCase.recorder)
		})
	}
}

func Test_Recorder_IsEmpty(t *testing.T) {
	testCases := map[string]struct {
		recorder *Recorder
		empty    bool
	}{
		"nil nodes": {
			recorder: &Recorder{},
			empty:    true,
		},
		"empty nodes": {
			recorder: &Recorder{
				nodes: []Node{},
			},
			empty: true,
		},
		"non-empty nodes": {
			recorder: &Recorder{
				nodes: []Node{{}},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			empty := testCase.recorder.IsEmpty()

			assert.Equal(t, testCase.empty, empty)
		})
	}
}
