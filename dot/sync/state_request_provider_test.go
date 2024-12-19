package sync

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/network/messages"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/trie"
	"github.com/ChainSafe/gossamer/pkg/trie/inmemory"
	"github.com/stretchr/testify/require"
)

func TestNewStateRequestProvider(t *testing.T) {
	targetHash := common.Hash{0x12, 0x23, 0x34}
	provider := NewStateRequestProvider(targetHash, trie.V0)

	require.NotNil(t, provider)

	require.Equal(t, [][]byte{}, provider.GetLastKeys())
	require.Equal(t, false, provider.IsCompleted())
}

func TestBuildRequest(t *testing.T) {
	targetHash := common.Hash{0x12, 0x23, 0x34}
	provider := NewStateRequestProvider(targetHash, trie.V0)

	req := provider.BuildRequest()

	require.NotNil(t, req)
	require.Equal(t, targetHash, req.Block)
	require.Equal(t, [][]byte{}, req.Start)
	require.True(t, req.NoProof)
}

func TestProcessResponse(t *testing.T) {
	tc := map[string]struct {
		response      *messages.StateResponse
		completed     bool
		expectedError error
	}{
		"empty_response_entries": {
			response:      &messages.StateResponse{},
			completed:     false,
			expectedError: errEmptyStateEntries,
		},
		"uncomplete_successful_response": {
			response: &messages.StateResponse{
				Entries: []messages.KeyValueStateEntry{
					{
						StateRoot: common.Hash{0x34, 0x45, 0x56},
						StateEntries: trie.Entries{
							{Key: []byte{0x01}, Value: []byte{0x11}},
							{Key: []byte{0x02}, Value: []byte{0x22}},
						},
						Complete: false,
					},
				},
			},
			completed:     false,
			expectedError: nil,
		},
		"complete_successful_response": {
			response: &messages.StateResponse{
				Entries: []messages.KeyValueStateEntry{
					{
						StateRoot: common.Hash{0x34, 0x45, 0x56},
						StateEntries: trie.Entries{
							{Key: []byte{0x01}, Value: []byte{0x11}},
							{Key: []byte{0x02}, Value: []byte{0x22}},
						},
						Complete: true,
					},
				},
			},
			completed:     true,
			expectedError: nil,
		},
	}

	for name, test := range tc {
		t.Run(name, func(t *testing.T) {
			targetHash := common.Hash{0x12, 0x23, 0x34}
			provider := NewStateRequestProvider(targetHash, trie.V0)

			completed, err := provider.ProcessResponse(test.response)

			require.Equal(t, test.completed, completed)
			require.Equal(t, test.expectedError, err)
		})
	}
}

func TestBuilTrie(t *testing.T) {
	targetHash := common.Hash{0x12, 0x23, 0x34}
	provider := NewStateRequestProvider(targetHash, trie.V0)

	trieEntries := trie.Entries{
		{Key: []byte("triekey1"), Value: []byte("trievalue1")},
		{Key: []byte("triekey2"), Value: []byte("trievalue2")},
		{Key: []byte("triekey3"), Value: []byte("trievalue3")},
		{Key: []byte("triekey4"), Value: []byte("trievalue4")},
	}

	expectedTrie := inmemory.NewEmptyTrie()
	for _, entry := range trieEntries {
		expectedTrie.Put(entry.Key, entry.Value)
	}

	response := &messages.StateResponse{
		Entries: []messages.KeyValueStateEntry{
			{
				StateRoot:    common.Hash{0x34, 0x45, 0x56},
				StateEntries: trieEntries,
				Complete:     true,
			},
		},
	}

	_, err := provider.ProcessResponse(response)
	require.Nil(t, err)

	resultTrie, err := provider.BuildTrie()
	require.Nil(t, err)
	require.Equal(t, expectedTrie.MustHash(), resultTrie.MustHash())
}
