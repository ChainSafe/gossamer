// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	"bytes"
	"testing"

	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const portsAmount = 200

// portQueue is a blocking port queue
type portQueue chan uint16

func (pq portQueue) put(p uint16) {
	pq <- p
}

func (pq portQueue) get() (port uint16) {
	port = <-pq
	return port
}

var availablePorts portQueue

func init() {
	availablePorts = make(chan uint16, portsAmount)
	const startAt = uint16(7500)
	for port := startAt; port < portsAmount+startAt; port++ {
		availablePorts.put(port)
	}
}

// availablePort is test helper function that gets an available port and release the same port after test ends
func availablePort(t *testing.T) uint16 {
	t.Helper()
	port := availablePorts.get()

	t.Cleanup(func() {
		availablePorts.put(port)
	})

	return port
}

// list of IPFS peers, for testing only
var TestPeers = []string{
	"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
	"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip4/104.236.76.40/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	"/ip6/2604:a880:1:20::203:d001/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
	"/ip6/2400:6180:0:d0::151:6001/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
	"/ip6/2604:a880:800:10::4a:5001/tcp/4001/ipfs/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
	"/ip6/2a03:b0c0:0:1010::23:1001/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
}

func TestStringToAddrInfo(t *testing.T) {
	for _, str := range TestPeers {
		pi, err := stringToAddrInfo(str)
		require.NoError(t, err)
		require.Equal(t, pi.ID.String(), str[len(str)-46:])
	}
}

func TestStringsToAddrInfos(t *testing.T) {
	pi, err := stringsToAddrInfos(TestPeers)
	require.NoError(t, err)

	for k, pi := range pi {
		require.Equal(t, pi.ID.String(), TestPeers[k][len(TestPeers[k])-46:])
	}
}

func TestGenerateKey(t *testing.T) {
	testDir := t.TempDir()

	keyA, err := generateKey(0, testDir)
	require.NoError(t, err)

	keyB, err := generateKey(0, testDir)
	require.NoError(t, err)
	require.NotEqual(t, keyA, keyB)

	keyC, err := generateKey(1, testDir)
	require.NoError(t, err)

	keyD, err := generateKey(1, testDir)
	require.NoError(t, err)
	require.Equal(t, keyC, keyD)
}

func TestReadLEB128ToUint64(t *testing.T) {
	tests := []struct {
		input  []byte
		output uint64
	}{
		{
			input:  []byte("\x02"),
			output: 2,
		},
		{
			input:  []byte("\x7F"),
			output: 127,
		},
		{
			input:  []byte("\x80\x01"),
			output: 128,
		},
		{
			input:  []byte("\x81\x01"),
			output: 129,
		},
		{
			input:  []byte("\x82\x01"),
			output: 130,
		},
		{
			input:  []byte("\xB9\x64"),
			output: 12857,
		},

		{
			input: []byte{'\xFF', '\xFF', '\xFF', '\xFF', '\xFF',
				'\xFF', '\xFF', '\xFF', '\xFF', '\x01'},
			output: 18446744073709551615,
		},
	}

	for _, tc := range tests {
		buf := new(bytes.Buffer)
		_, err := buf.Write(tc.input)
		require.NoError(t, err)

		ret, _, err := ReadLEB128ToUint64(buf)
		require.NoError(t, err)
		require.Equal(t, tc.output, ret)
	}
}

func TestInvalidLeb128(t *testing.T) {
	input := []byte{'\xFF', '\xFF', '\xFF', '\xFF', '\xFF',
		'\xFF', '\xFF', '\xFF', '\xFF', '\xFF', '\x01'}
	buf := new(bytes.Buffer)
	_, err := buf.Write(input)
	require.NoError(t, err)

	_, _, err = ReadLEB128ToUint64(buf)
	require.Error(t, err)
}

func TestReadStream(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		maxSize         uint64
		bufPointer      *[]byte
		buildStreamMock func(ctrl *gomock.Controller) libp2pnetwork.Stream
		wantErr         error
		errString       string
		expectedOutput  int
		expectedBuf     []byte
	}{
		"nil_stream": {
			buildStreamMock: func(ctrl *gomock.Controller) libp2pnetwork.Stream {
				return nil
			},
			wantErr:        ErrNilStream,
			errString:      "nil stream",
			expectedOutput: 0,
		},

		"invalid_leb128": {
			buildStreamMock: func(ctrl *gomock.Controller) libp2pnetwork.Stream {
				input := []byte{'\xFF', '\xFF', '\xFF', '\xFF', '\xFF',
					'\xFF', '\xFF', '\xFF', '\xFF', '\xFF', '\x01'}

				invalidLeb128Buf := new(bytes.Buffer)
				_, err := invalidLeb128Buf.Write(input)
				require.NoError(t, err)

				streamMock := NewMockStream(ctrl)

				streamMock.EXPECT().Read([]byte{0}).
					DoAndReturn(func(buf any) (n, err any) {
						return invalidLeb128Buf.Read(buf.([]byte))
					}).MaxTimes(10)

				return streamMock
			},
			bufPointer:     &[]byte{0},
			expectedOutput: 10, // read all the bytes in the invalidLeb128Buf
			wantErr:        ErrInvalidLEB128EncodedData,
			errString:      "failed to read length: invalid LEB128 encoded data",
		},

		"zero_length": {
			buildStreamMock: func(ctrl *gomock.Controller) libp2pnetwork.Stream {
				input := []byte{'\x00'}

				streamBuf := new(bytes.Buffer)
				_, err := streamBuf.Write(input)
				require.NoError(t, err)

				streamMock := NewMockStream(ctrl)

				streamMock.EXPECT().Read([]byte{0}).
					DoAndReturn(func(buf any) (n, err any) {
						return streamBuf.Read(buf.([]byte))
					})

				return streamMock
			},
			bufPointer:     &[]byte{0},
			expectedOutput: 0,
		},

		"length_greater_than_buf_increase_buf_size": {
			buildStreamMock: func(ctrl *gomock.Controller) libp2pnetwork.Stream {
				input := []byte{0xa, //size 0xa == 10
					0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, // actual data
				}

				streamBuf := new(bytes.Buffer)
				_, err := streamBuf.Write(input)
				require.NoError(t, err)

				streamMock := NewMockStream(ctrl)

				streamMock.EXPECT().Read([]byte{0}).
					DoAndReturn(func(buf any) (n, err any) {
						return streamBuf.Read(buf.([]byte))
					})

				streamMock.EXPECT().Read(make([]byte, 10)).
					DoAndReturn(func(buf any) (n, err any) {
						return streamBuf.Read(buf.([]byte))
					})

				return streamMock
			},
			bufPointer:     &[]byte{0}, // a buffer with size 1
			expectedBuf:    []byte{0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1},
			expectedOutput: 10,
			maxSize:        11,
		},

		"length_greater_than_max_size": {
			buildStreamMock: func(ctrl *gomock.Controller) libp2pnetwork.Stream {
				input := []byte{0xa, //size 0xa == 10
					0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, // actual data
				}

				streamBuf := new(bytes.Buffer)
				_, err := streamBuf.Write(input)
				require.NoError(t, err)

				streamMock := NewMockStream(ctrl)

				streamMock.EXPECT().Read([]byte{0}).
					DoAndReturn(func(buf any) (n, err any) {
						return streamBuf.Read(buf.([]byte))
					})

				return streamMock
			},
			bufPointer: &[]byte{0}, // a buffer with size 1
			wantErr:    ErrGreaterThanMaxSize,
			errString:  "greater than maximum size: max 9, got 10",
			maxSize:    9,
		},
	}

	for tname, tt := range cases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			stream := tt.buildStreamMock(ctrl)

			n, err := readStream(stream, tt.bufPointer, tt.maxSize)
			require.Equal(t, tt.expectedOutput, n)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.errString != "" {
				require.EqualError(t, err, tt.errString)
			}

			if tt.expectedBuf != nil {
				require.Equal(t, tt.expectedBuf, *tt.bufPointer)
			}
		})
	}
}
