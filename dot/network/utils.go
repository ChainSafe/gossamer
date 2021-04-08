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

package network

import (
	"bufio"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	mrand "math/rand"
	"os"
	"path"
	"path/filepath"

	"github.com/libp2p/go-libp2p-core/crypto"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
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
		r = mrand.New(mrand.NewSource(seed)) //nolint
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

// makeDir makes directory if directory does not already exist
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

func uint64ToLEB128(in uint64) []byte {
	out := []byte{}
	for {
		b := uint8(in & 0x7f)
		in >>= 7
		if in != 0 {
			b |= 0x80
		}
		out = append(out, b)
		if in == 0 {
			break
		}
	}
	return out
}

func readLEB128ToUint64(r *bufio.Reader) (uint64, error) {
	var out uint64
	var shift uint
	for {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}

		out |= uint64(0x7F&b) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return out, nil
}

// readStream reads from the stream into the given buffer, returning the number of bytes read
func readStream(stream libp2pnetwork.Stream, buf []byte) (int, error) {
	if stream == nil {
		return 0, errors.New("stream is nil")
	}

	r := bufio.NewReader(stream)

	var (
		tot int
	)

	length, err := readLEB128ToUint64(r)
	if err == io.EOF {
		return 0, err
	} else if err != nil {
		return 0, err // TODO: return bytes read from readLEB128ToUint64
	}

	if length == 0 {
		return 0, err // TODO: return bytes read from readLEB128ToUint64
	}

	// TODO: check if length > len(buf), if so probably log.Crit
	if length > maxBlockResponseSize {
		logger.Warn("received message with size greater than maxBlockResponseSize, discarding", "length", length)
		for {
			_, err = r.Discard(int(maxBlockResponseSize))
			if err != nil {
				break
			}
		}
		return 0, fmt.Errorf("message size greater than maximum: got %d", length)
	}

	tot = 0
	for i := 0; i < maxReads; i++ {
		n, err := r.Read(buf[tot:])
		if err != nil {
			return n + tot, err
		}

		tot += n
		if tot == int(length) {
			break
		}
	}

	if tot != int(length) {
		return tot, fmt.Errorf("failed to read entire message: expected %d bytes", length)
	}

	return tot, nil
}
