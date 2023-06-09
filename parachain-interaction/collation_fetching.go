package parachaininteraction

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// pub struct CollationFetchingRequest {
// 	/// Relay parent we want a collation for.
// 	pub relay_parent: Hash,
// 	/// The `ParaId` of the collation.
// 	pub para_id: ParaId,
// }

type CollationFetchingRequest struct {
	RelayParent common.Hash
	ParaID      uint32
}

// Encode returns the encoded CollationFetchingRequest
func (cfr *CollationFetchingRequest) Encode() ([]byte, error) {
	return scale.Marshal(cfr)
}

// /// Responses as sent by collators.
// #[derive(Debug, Clone, Encode, Decode)]
// pub enum CollationFetchingResponse {
// 	/// Deliver requested collation.
// 	#[codec(index = 0)]
// 	Collation(CandidateReceipt, PoV),
// }

func RequestCollation(p2pHost host.Host, to peer.ID, genesisHash common.Hash) {
	DoRequest(context.Background(), p2pHost, to, CollationFetchingV1, "", genesisHash, &CollationFetchingRequest{
		// TODO: where to get true values of relayparent or paraid
		RelayParent: common.MustHexToHash("0xb6d36a6766363567d2a385c8b5f9bd93b223b8f42e54aa830270edcf375f4d63"),
		ParaID:      2000,
	})
}

func DoRequest(ctx context.Context, p2pHost host.Host, to peer.ID, protocolName ReqProtocolName, forkID string, genesisHash common.Hash, req network.Message) (network.Message, error) {
	protocolID := protocol.ID(GenerateReqProtocolName(protocolName, forkID, genesisHash))

	p2pHost.ConnManager().Protect(to, "")
	defer p2pHost.ConnManager().Unprotect(to, "")

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	stream, err := p2pHost.NewStream(ctx, to, protocolID)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := stream.Close()
		if err != nil {
			logger.Warnf("failed to close stream: %s", err)
		}
	}()

	if err = writeToStream(stream, req); err != nil {
		return nil, err
	}

	return ReceiveResponse(stream)

}

func ReceiveResponse(stream libp2pnetwork.Stream) (network.Message, error) {
	buf := make([]byte, maxResponseSize)
	n, err := readStream(stream, &buf, maxResponseSize)
	if err != nil {
		return nil, fmt.Errorf("read stream error: %w", err)
	}

	if n == 0 {
		return nil, fmt.Errorf("received empty message")
	}

	fmt.Println("some parachain related response", buf)
	// msg := new(BlockResponseMessage)
	// err = msg.Decode(buf[:n])
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to decode block response: %w", err)
	// }

	return nil, nil

}

func writeToStream(s libp2pnetwork.Stream, msg network.Message) error {
	encMsg, err := msg.Encode()
	if err != nil {
		return err
	}

	msgLen := uint64(len(encMsg))
	lenBytes := uint64ToLEB128(msgLen)
	encMsg = append(lenBytes, encMsg...)

	_, err = s.Write(encMsg)
	if err != nil {
		return err
	}

	return nil
}

func uint64ToLEB128(in uint64) []byte {
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

// readStream reads from the stream into the given buffer, returning the number of bytes read
func readStream(stream libp2pnetwork.Stream, bufPointer *[]byte, maxSize uint64) (int, error) {
	if stream == nil {
		return 0, errors.New("stream is nil")
	}

	var (
		tot int
	)

	buf := *bufPointer
	length, bytesRead, err := readLEB128ToUint64(stream, buf[:1])
	if err != nil {
		return bytesRead, fmt.Errorf("failed to read length: %w", err)
	}

	if length == 0 {
		return 0, nil // msg length of 0 is allowed, for example transactions handshake
	}

	if length > uint64(len(buf)) {
		extraBytes := int(length) - len(buf)
		*bufPointer = append(buf, make([]byte, extraBytes)...) // TODO #2288 use bytes.Buffer instead
		logger.Warnf("received message with size %d greater than allocated message buffer size %d", length, len(buf))
	}

	if length > maxSize {
		logger.Warnf("received message with size %d greater than max size %d, closing stream", length, maxSize)
		return 0, fmt.Errorf("message size greater than maximum: got %d", length)
	}

	tot = 0
	for i := 0; i < maxReads; i++ {
		n, err := stream.Read(buf[tot:])
		if err != nil {
			return n + tot, err
		}

		tot += n
		if tot == int(length) {
			break
		}
	}

	if tot != int(length) {
		return tot, fmt.Errorf("failed to read entire message: expected %d bytes, received %d bytes", length, tot)
	}

	return tot, nil
}

func readLEB128ToUint64(r io.Reader, buf []byte) (uint64, int, error) {
	if len(buf) == 0 {
		return 0, 0, errors.New("buffer has length 0")
	}

	var out uint64
	var shift uint

	maxSize := 10 // Max bytes in LEB128 encoding of uint64 is 10.
	bytesRead := 0

	for {
		n, err := r.Read(buf[:1])
		if err != nil {
			return 0, bytesRead, err
		}

		bytesRead += n

		b := buf[0]
		out |= uint64(0x7F&b) << shift
		if b&0x80 == 0 {
			break
		}

		maxSize--
		if maxSize == 0 {
			return 0, bytesRead, fmt.Errorf("invalid LEB128 encoded data")
		}

		shift += 7
	}
	return out, bytesRead, nil
}
