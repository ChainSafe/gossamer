// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more detailg.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/stretchr/testify/require"
)

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
		require.Equal(t, pi.ID.Pretty(), str[len(str)-46:])
	}
}

func TestStringsToAddrInfos(t *testing.T) {
	pi, err := stringsToAddrInfos(TestPeers)
	require.NoError(t, err)

	for k, pi := range pi {
		require.Equal(t, pi.ID.Pretty(), TestPeers[k][len(TestPeers[k])-46:])
	}
}

func TestGenerateKey(t *testing.T) {
	testDir := utils.NewTestDir(t)
	defer utils.RemoveTestDir(t)

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
	}

	for _, tc := range tests {
		b := make([]byte, 2)
		buf := new(bytes.Buffer)
		_, err := buf.Write(tc.input)
		require.NoError(t, err)

		ret, err := readLEB128ToUint64(buf, b[:1])
		require.NoError(t, err)
		require.Equal(t, tc.output, ret)
	}
}
