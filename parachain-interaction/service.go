package parachaininteraction

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachains"))
var maxReads = 256
var maxResponseSize uint64 = 1024 * 1024 * 16 // 16mb

// Notes:
/*
There are two types of peersets, validation and collation

Network Message types her https://paritytech.github.io/polkadot/book/types/network.html#validation-v1

Messages over Validation Protocol
enum ValidationProtocolV1 {
    ApprovalDistribution(ApprovalDistributionV1Message),
    AvailabilityDistribution(AvailabilityDistributionV1Message),
    AvailabilityRecovery(AvailabilityRecoveryV1Message),
    BitfieldDistribution(BitfieldDistributionV1Message),
    PoVDistribution(PoVDistributionV1Message),
    StatementDistribution(StatementDistributionV1Message),
}

Messages over Collation Protocol
enum CollationProtocolV1 {
    CollatorProtocol(CollatorProtocolV1Message),
}

*/
const MaxValidationMessageSize uint64 = 100 * 1024
const MaxCollationMessageSize uint64 = 100 * 1024

type Service struct {
	Network Network
}

func NewService(net Network, genesisHash common.Hash) (*Service, error) {

	// TODO: Change this and give different message type for each protocol
	validationMsgType := byte(10)
	collationMsgType := byte(11)
	// TODO: Where do I get forkID and version from from?
	forkID := ""
	var version uint32 = 1

	validationProtocolID := GeneratePeersetProtocolName(ValidationProtocol, forkID, genesisHash, version)
	// register validation protocol
	// TODO: It seems like handshake is None, but be sure of it.
	err := net.RegisterNotificationsProtocol(
		protocol.ID(validationProtocolID),
		validationMsgType,
		func() (network.Handshake, error) {
			return nil, nil
		},
		func(_ []byte) (network.Handshake, error) {
			return nil, nil
		},
		func(_ peer.ID, _ network.Handshake) error {
			return nil
		},
		decodeValidationMessage,
		handleValidationMessage,
		nil,
		MaxValidationMessageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("registering validation protocol: %w", err)
	}

	collationProtocolID := GeneratePeersetProtocolName(CollationProtocol, forkID, genesisHash, version)
	// register collation protocol
	// TODO: It seems like handshake is None, but be sure of it.
	err = net.RegisterNotificationsProtocol(
		protocol.ID(collationProtocolID),
		collationMsgType,
		func() (network.Handshake, error) {
			return nil, nil
		},
		func(_ []byte) (network.Handshake, error) {
			return nil, nil
		},
		func(_ peer.ID, _ network.Handshake) error {
			return nil
		},
		decodeCollationMessage,
		handleCollationMessage,
		nil,
		MaxCollationMessageSize,
	)
	if err != nil {
		return nil, fmt.Errorf("registering collation protocol: %w", err)
	}

	// TODO: Add request response protocol for collation fetching
	protocolID := protocol.ID(GenerateReqProtocolName(CollationFetchingV1, forkID, genesisHash))

	rrp := net.GetRequestResponseProtocol(protocolID, requestTimeout, maxResponseSize)

	return &Service{
		Network: net,
	}, nil
}

// Start starts the Handler
func (Service) Start() error {
	return nil
}

// Stop stops the Handler
func (Service) Stop() error {
	return nil
}

// Network is the interface required by GRANDPA for the network
type Network interface {
	GossipMessage(msg network.NotificationsMessage)
	SendMessage(to peer.ID, msg network.NotificationsMessage) error
	RegisterNotificationsProtocol(sub protocol.ID,
		messageID byte,
		handshakeGetter network.HandshakeGetter,
		handshakeDecoder network.HandshakeDecoder,
		handshakeValidator network.HandshakeValidator,
		messageDecoder network.MessageDecoder,
		messageHandler network.NotificationsMessageHandler,
		batchHandler network.NotificationsMessageBatchHandler,
		maxSize uint64,
	) error
	GetRequestResponseProtocol(protocolID protocol.ID, requestTimeout time.Duration, maxResponseSize uint64) *network.RequestResponseProtocol
}

func decodeValidationMessage(in []byte) (network.NotificationsMessage, error) {
	// TODO: add things
	fmt.Println("We got a validation message", in)
	return nil, nil
}

func handleValidationMessage(peerID peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a validation message", msg)
	return false, nil
}

func decodeCollationMessage(in []byte) (network.NotificationsMessage, error) {
	fmt.Println("We got a collation message", in)
	// TODO: add things
	return nil, nil
}

func handleCollationMessage(peerID peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a collation message", msg)
	return false, nil
}

var (
	requestTimeout = time.Second * 20
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

// 	// DoBlockRequest sends a request to the given peer.
// // If a response is received within a certain time period, it is returned,
// // otherwise an error is returned.
// func (s *Service) DoBlockRequest(to peer.ID, req *BlockRequestMessage) (*BlockResponseMessage, error) {
// 	fullSyncID := s.host.protocolID + syncID

// 	s.host.p2pHost.ConnManager().Protect(to, "")
// 	defer s.host.p2pHost.ConnManager().Unprotect(to, "")

// 	ctx, cancel := context.WithTimeout(s.ctx, blockRequestTimeout)
// 	defer cancel()

// 	stream, err := s.host.p2pHost.NewStream(ctx, to, fullSyncID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	defer func() {
// 		err := stream.Close()
// 		if err != nil {
// 			logger.Warnf("failed to close stream: %s", err)
// 		}
// 	}()

// 	if err = s.host.writeToStream(stream, req); err != nil {
// 		return nil, err
// 	}

// 	return s.receiveBlockResponse(stream)
// }
