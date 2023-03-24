package sync

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common/variadic"
	"github.com/libp2p/go-libp2p/core/peer"
)

const maxResponseSize = 128

type syncerWorkerByNumber struct {
	locker    sync.RWMutex
	available bool
	done      chan struct{}

	peerID peer.ID

	startNumber  uint
	targetNumber uint

	dataToRequest byte
	direction     network.SyncDirection
	network       Network
	retryNumber   uint8
}

func newSyncerWorkerByNumber(peerID peer.ID, start,
	target uint, requestData byte, direction network.SyncDirection,
	retryNumber uint8, network Network) *syncerWorkerByNumber {
	return &syncerWorkerByNumber{
		available:     false,
		done:          make(chan struct{}),
		peerID:        peerID,
		startNumber:   start,
		targetNumber:  target,
		direction:     direction,
		dataToRequest: requestData,
		retryNumber:   retryNumber,
		network:       network,
	}
}

func (s *syncerWorkerByNumber) String() string {
	template := "worker(%s): {available: %v, start: %d, target %d, direction: %s, retries: %d}"
	return fmt.Sprintf(template, s.peerID, s.available, s.startNumber, s.targetNumber, s.direction, s.retryNumber)
}

func (s *syncerWorkerByNumber) Dispatch() (blockDataResults []*types.BlockData, err error) {
	s.locker.Lock()
	defer func() {
		if err == nil {
			s.available = true
		}

		close(s.done)
		s.locker.Unlock()
	}()

	logger.Infof("[WAITING] dispatching sync worker id %s, "+
		"start number %d, target number %d, "+
		"request data %d, direction %s",
		s.peerID,
		s.startNumber, s.targetNumber,
		s.dataToRequest, s.direction)

	request, err := s.toRequest()
	if err != nil {
		return nil, fmt.Errorf("while creating the request: %w", err)
	}

	response, err := s.network.DoBlockRequest(s.peerID, request)
	if err != nil {
		return nil, fmt.Errorf("while executing the request: %w", err)
	}

	err = validateResponse(request, response)
	if err != nil {
		return nil, fmt.Errorf("validating response: %w", err)
	}

	logger.Infof("[SUCESSFULL] dispatching sync worker id %s, "+
		"start number %d, target number %d, "+
		"request data %d, direction %s",
		s.peerID,
		s.startNumber, s.targetNumber,
		s.dataToRequest, s.direction)

	return response.BlockData, nil
}

var errEmptyBlockData = errors.New("empty block data")
var errResponseIsNotChain = errors.New("block response does not form a chain")

func validateResponse(req *network.BlockRequestMessage,
	resp *network.BlockResponseMessage) error {
	if resp == nil || len(resp.BlockData) == 0 {
		return errEmptyBlockData
	}

	var (
		prev, curr *types.Header
		err        error
	)

	for i, bd := range resp.BlockData {
		if err = validateBlockData(req, bd); err != nil {
			return err
		}

		curr = bd.Header
		if i == 0 {
			prev = curr
			continue
		}

		// otherwise, check that this response forms a chain
		// ie. curr's parent hash is hash of previous header, and curr's number is previous number + 1
		if prev.Hash() != curr.ParentHash || curr.Number != prev.Number+1 {
			return errResponseIsNotChain
		}

		prev = curr
	}

	return nil
}

var errNilBlockData = errors.New("block data is nil")
var errNilHeaderInResponse = errors.New("expected header, received none")
var errNilBodyInResponse = errors.New("expected body, received none")

func validateBlockData(req *network.BlockRequestMessage, bd *types.BlockData) error {
	if bd == nil {
		return errNilBlockData
	}

	requestedData := req.RequestedData

	if (requestedData&network.RequestedDataHeader) == 1 && bd.Header == nil {
		return errNilHeaderInResponse
	}

	if (requestedData&network.RequestedDataBody>>1) == 1 && bd.Body == nil {
		return fmt.Errorf("%w: hash=%s", errNilBodyInResponse, bd.Hash)
	}

	return nil
}

func (s *syncerWorkerByNumber) toRequest() (*network.BlockRequestMessage, error) {
	diff := s.targetNumber - s.startNumber
	// start and end block are the same, just request 1 block
	if diff == 0 {
		diff = 1
	}

	start := variadic.MustNewUint32OrHash(uint32(s.startNumber))
	max := new(uint32)
	if diff > maxBlocksToRequest {
		*max = maxBlocksToRequest
	} else {
		*max = uint32(diff)
	}

	return &network.BlockRequestMessage{
		RequestedData: s.dataToRequest,
		StartingBlock: *start,
		Direction:     s.direction,
		Max:           max,
	}, nil
}

func (s *syncerWorkerByNumber) Stop() error { return nil }
