// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package network

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"path"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	// maxBlockRequestSize              uint64 = 1024 * 1024      // 1mb
	MaxBlockResponseSize uint64 = 1024 * 1024 * 16 // 16mb
	// MaxGrandpaNotificationSize is maximum size for a grandpa notification message.
	MaxGrandpaNotificationSize       uint64 = 1024 * 1024      // 1mb
	maxTransactionsNotificationSize  uint64 = 1024 * 1024 * 16 // 16mb
	maxBlockAnnounceNotificationSize uint64 = 1024 * 1024      // 1mb

)

func isInbound(stream libp2pnetwork.Stream) bool {
	return stream.Stat().Direction == libp2pnetwork.DirInbound
}

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
		r = mrand.New(mrand.NewSource(seed)) //nolint:gosec
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
	keyData, err := os.ReadFile(filepath.Clean(pth))
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
	f, err := os.Create(filepath.Clean(pth))
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

func Uint64ToLEB128(in uint64) []byte {
	var out []byte
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

func ReadLEB128ToUint64(r io.Reader) (uint64, int, error) {
	var out uint64
	var shift uint

	maxSize := 10 // Max bytes in LEB128 encoding of uint64 is 10.
	bytesRead := 0

	for {
		// read a sinlge byte
		singleByte := []byte{0}
		n, err := r.Read(singleByte)
		if err != nil {
			return 0, bytesRead, err
		}

		bytesRead += n

		b := singleByte[0]
		out |= uint64(0x7F&b) << shift
		if b&0x80 == 0 {
			break
		}

		maxSize--
		if maxSize == 0 {
			return 0, bytesRead, ErrInvalidLEB128EncodedData
		}

		shift += 7
	}
	return out, bytesRead, nil
}

// readStream reads from the stream into the given buffer, returning the number of bytes read
func readStream(stream libp2pnetwork.Stream, bufPointer *[]byte, maxSize uint64) (tot int, err error) {
	if stream == nil {
		return 0, ErrNilStream
	}

	length, bytesRead, err := ReadLEB128ToUint64(stream)
	if err != nil {
		return bytesRead, fmt.Errorf("failed to read length: %w", err)
	}

	if length == 0 {
		return 0, nil // msg length of 0 is allowed, for example transactions handshake
	}

	buf := *bufPointer
	if length > uint64(len(buf)) {
		logger.Warnf("received message with size %d greater than allocated message buffer size %d", length, len(buf))
		extraBytes := int(length) - len(buf)
		*bufPointer = append(buf, make([]byte, extraBytes)...)
		buf = *bufPointer
	}

	if length > maxSize {
		logger.Warnf("received message with size %d greater than max size %d, closing stream", length, maxSize)
		return 0, fmt.Errorf("%w: max %d, got %d", ErrGreaterThanMaxSize, maxSize, length)
	}

	for tot < int(length) {
		n, err := stream.Read(buf[tot:])
		if err != nil {
			return n + tot, err
		}

		tot += n
	}

	if tot != int(length) {
		return tot, fmt.Errorf("%w: expected %d bytes, received %d bytes", ErrFailedToReadEntireMessage, length, tot)
	}

	return tot, nil
}
