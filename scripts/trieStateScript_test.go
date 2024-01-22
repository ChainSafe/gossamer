package main

import (
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/require"
)

// This is fake data used just for testing purposes
var testStateData = []string{"0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000", "0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000", "0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000", "0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000", "0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000", "0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000", "0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000", "0x801cb6f36e027abb2091cfb5110ab5087faacf00b9b41fda7a9268821c2a2b3e4ca404d43593c715fdd31c61141abd04a99fd6822c8558854ccde39a5684e7a56da27d0100000000000000"} //nolint

func clean(t *testing.T, file string) {
	t.Helper()
	err := os.Remove(file)
	require.NoError(t, err)
}

func Test_writeTrieState(t *testing.T) {
	writeTrieState(testStateData, "westendDevTestState.json")
	_, err := os.Stat("./westendDevTestState.json")
	require.NoError(t, err)

	clean(t, "westendDevTestState.json")
}

func Test_compareStateRoots(t *testing.T) {
	type args struct {
		response          modules.StateTrieResponse
		expectedStateRoot common.Hash
		trieVersion       int
	}
	tests := []struct {
		name        string
		args        args
		shouldPanic bool
	}{
		{
			name: "happy path",
			args: args{
				response:          testStateData,
				expectedStateRoot: common.MustHexToHash("0x3b1863ff981a31864be76037e4cf5c927b937dd8a8e1e25494128da7a95b5cdf"),
				trieVersion:       0,
			},
		},
		{
			name: "invalid trie version",
			args: args{
				response:          testStateData,
				expectedStateRoot: common.MustHexToHash("0x6120d3afde6c139305bd7c0dcf50bdff5b620203e00c7491b2c30f95dccacc32"),
				trieVersion:       21,
			},
			shouldPanic: true,
		},
		{
			name: "hashes do not match",
			args: args{
				response:          testStateData,
				expectedStateRoot: common.MustHexToHash("0x01"),
				trieVersion:       21,
			},
			shouldPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				require.Panics(t,
					func() {
						compareStateRoots(tt.args.response, tt.args.expectedStateRoot, tt.args.trieVersion)
					},
					"The code did not panic")
			} else {
				compareStateRoots(tt.args.response, tt.args.expectedStateRoot, tt.args.trieVersion)
			}
		})
	}
}

func Test_cli(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no arguments",
		},
		{
			name: "to few arguments",
			args: []string{"0x01"},
		},
		{
			name: "invalid formatting for block hash",
			args: []string{"hello", "output.json"},
		},
		{
			name: "no trie version",
			args: []string{"0x01", "output.json", "0x01"},
		},
		{
			name: "invalid formatting for root hash",
			args: []string{"0x01", "output.json", "hello", "1"},
		},
		{
			name: "invalid trie version",
			args: []string{"0x01", "output.json", "0x01", "hello"},
		},
		{
			name: "to many arguments",
			args: []string{"0x01", "output.json", "0x01", "1", "0x01"},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			require.Panics(t, func() { main() }, "The code did not panic")
		})
	}
}
