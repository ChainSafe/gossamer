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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package p2p

import (
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"os"
	"path"
	"path/filepath"

	"github.com/ChainSafe/gossamer/common"
	"github.com/filecoin-project/go-leb128"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// stringToAddrInfos converts a single string peer id to AddrInfo
func stringToAddrInfo(s string) (peer.AddrInfo, error) {
	maddr, err := multiaddr.NewMultiaddr(s)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	p, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	return *p, err
}

// stringsToAddrInfos converts a string of peer ids to AddrInfo
func stringsToAddrInfos(peers []string) ([]peer.AddrInfo, error) {
	pinfos := make([]peer.AddrInfo, len(peers))
	for i, p := range peers {
		p, err := stringToAddrInfo(p)
		if err != nil {
			return nil, err
		}
		pinfos[i] = p
	}
	return pinfos, nil
}

// generateKey generates an ed25519 private key and writes it to the data directory
// If the seed is zero, we use real cryptographic randomness. Otherwise, we use a
// deterministic randomness source to make keys the same across multiple runs.
func generateKey(seed int64, fp string) (crypto.PrivKey, error) {
	var r io.Reader
	if seed == 0 {
		r = crand.Reader
	} else {
		r = mrand.New(mrand.NewSource(seed))
	}
	key, _, err := crypto.GenerateEd25519Key(r)
	if err != nil {
		return nil, err
	}
	if seed == 0 {
		if err = makeDir(fp); err != nil {
			return nil, err
		}
		if err = saveKey(key, fp); err != nil {
			return nil, err
		}
	}
	return key, nil
}

// loadKey attempts to load a private key from the provided filepath
func loadKey(fp string) (crypto.PrivKey, error) {
	pth := path.Join(filepath.Clean(fp), DefaultKeyFile)
	if _, err := os.Stat(pth); os.IsNotExist(err) {
		return nil, nil
	}
	keyData, err := ioutil.ReadFile(filepath.Clean(pth))
	if err != nil {
		return nil, err
	}
	dec := make([]byte, hex.DecodedLen(len(keyData)))
	_, err = hex.Decode(dec, keyData)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalEd25519PrivateKey(dec)
}

// makeDir creates `.gossamer` if directory does not already exist
func makeDir(fp string) error {
	_, e := os.Stat(fp)
	if os.IsNotExist(e) {
		e = os.Mkdir(fp, os.ModePerm)
		if e != nil {
			return e
		}
	}
	return e
}

// saveKey attempts to save a private key to the provided filepath
func saveKey(priv crypto.PrivKey, fp string) (err error) {
	pth := path.Join(filepath.Clean(fp), DefaultKeyFile)
	f, err := os.Create(pth)
	if err != nil {
		return err
	}
	raw, err := priv.Raw()
	if err != nil {
		return err
	}
	enc := make([]byte, hex.EncodedLen(len(raw)))
	hex.Encode(enc, raw)
	if _, err = f.Write(enc); err != nil {
		return err
	}
	return f.Close()
}

// encodeMessageLEB128 applies leb128 variable-length encoding
func encodeMessageLEB128(encMsg []byte) []byte {
	out := leb128.FromUInt64(uint64(len(encMsg)))
	return append(out, encMsg...)
}

// decodeMessage decodes the message based on message type
func decodeMessage(r io.Reader) (m Message, err error) {
	msgType, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	switch msgType {
	case StatusMsgType:
		m = new(StatusMessage)
		err = m.Decode(r)
	case BlockRequestMsgType:
		m = new(BlockRequestMessage)
		err = m.Decode(r)
	case BlockResponseMsgType:
		m = new(BlockResponseMessage)
		err = m.Decode(r)
	case BlockAnnounceMsgType:
		m = new(BlockAnnounceMessage)
		err = m.Decode(r)
	case TransactionMsgType:
		m = new(TransactionMessage)
		err = m.Decode(r)
	default:
		return nil, errors.New("unsupported message type")
	}

	return m, err
}
