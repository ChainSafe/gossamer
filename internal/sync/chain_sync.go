package sync

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/dot/peerset"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	rtstorage "github.com/ChainSafe/gossamer/lib/runtime/storage"
	"github.com/libp2p/go-libp2p/core/peer"
)

var logger = log.NewFromGlobal(log.AddContext("internal", "sync"))

type Network interface {
	// DoBlockRequest sends a request to the given peer.
	// If a response is received within a certain time period,
	// it is returned, otherwise an error is returned.
	DoBlockRequest(to peer.ID, req *network.BlockRequestMessage) (*network.BlockResponseMessage, error)

	// Peers returns a list of currently connected peers
	Peers() []common.PeerInfo

	TotalConnectedPeers() []peer.ID

	// ReportPeer reports peer based on the peer behaviour.
	ReportPeer(change peerset.ReputationChange, p peer.ID)
}

type BlockState interface {
	BestBlockHeader() (*types.Header, error)
	BestBlockNumber() (number uint, err error)
	CompareAndSetBlockData(bd *types.BlockData) error
	HasBlockBody(hash common.Hash) (bool, error)
	GetBlockBody(common.Hash) (*types.Body, error)
	GetHeader(common.Hash) (*types.Header, error)
	HasHeader(hash common.Hash) (bool, error)
	Range(startHash, endHash common.Hash) (hashes []common.Hash, err error)
	RangeInMemory(start, end common.Hash) ([]common.Hash, error)
	GetReceipt(common.Hash) ([]byte, error)
	GetMessageQueue(common.Hash) ([]byte, error)
	GetJustification(common.Hash) ([]byte, error)
	SetJustification(hash common.Hash, data []byte) error
	AddBlockToBlockTree(block *types.Block) error
	GetHashByNumber(blockNumber uint) (common.Hash, error)
	GetBlockByHash(common.Hash) (*types.Block, error)
	GetRuntime(blockHash common.Hash) (runtime state.Runtime, err error)
	StoreRuntime(blockHash common.Hash, runtime state.Runtime)
	GetHighestFinalisedHeader() (*types.Header, error)
	GetFinalisedNotifierChannel() chan *types.FinalisationInfo
	GetHeaderByNumber(num uint) (*types.Header, error)
	GetAllBlocksAtNumber(num uint) ([]common.Hash, error)
	IsDescendantOf(parent, child common.Hash) (bool, error)
}

// StorageState is the interface for the storage state
type StorageState interface {
	TrieState(root *common.Hash) (*rtstorage.TrieState, error)
	sync.Locker
}

// BlockImportHandler is the interface for the handler of newly imported blocks
type BlockImportHandler interface {
	HandleBlockImport(block *types.Block, state *rtstorage.TrieState, announce bool) error
}

// BabeVerifier deals with BABE block verification
type BabeVerifier interface {
	VerifyBlock(header *types.Header) error
}

// Telemetry is the telemetry client to send telemetry messages.
type Telemetry interface {
	SendMessage(msg json.Marshaler)
}

// FinalityGadget implements justification verification functionality
type FinalityGadget interface {
	VerifyBlockJustification(common.Hash, []byte) error
}

// TransactionState is the interface for transaction queue methods
type TransactionState interface {
	RemoveExtrinsic(ext types.Extrinsic)
}

// Config is the configuration for the sync Service.
type Config struct {
	LogLvl             log.Level
	Network            Network
	BlockState         BlockState
	StorageState       StorageState
	FinalityGadget     FinalityGadget
	TransactionState   TransactionState
	BlockImportHandler BlockImportHandler
	BabeVerifier       BabeVerifier
	MinPeers, MaxPeers int
	SlotDuration       time.Duration
	Telemetry          Telemetry
}

type Service struct {
	bootstrapSyncer *bootstrapSyncer
}

// NewService returns a new *sync.Service
func NewService(cfg *Config) (*Service, error) {
	logger.Patch(log.SetLevel(cfg.LogLvl))

	bootstrapSyncer := &bootstrapSyncer{
		blockState:         cfg.BlockState,
		storageState:       cfg.StorageState,
		blockImportHandler: cfg.BlockImportHandler,
		network:            cfg.Network,
		babeVerifier:       cfg.BabeVerifier,
		transactionState:   cfg.TransactionState,
		telemetry:          cfg.Telemetry,
		finalityGadget:     cfg.FinalityGadget,
		offset:             0,
	}

	return &Service{
		bootstrapSyncer: bootstrapSyncer,
	}, nil
}

func (s *Service) HandleBlockAnnounceHandshake(from peer.ID, msg *network.BlockAnnounceHandshake) error {
	//fmt.Printf("===> receiving block announce from %s\n", from)
	return nil
}

// HandleBlockAnnounce is called upon receipt of a BlockAnnounceMessage to process it.
// If a request needs to be sent to the peer to retrieve the full block, this function will return it.
func (s *Service) HandleBlockAnnounce(from peer.ID, msg *network.BlockAnnounceMessage) error {
	return nil
}

// IsSynced exposes the internal synced state
func (s *Service) IsSynced() bool { return false }

// CreateBlockResponse is called upon receipt of a BlockRequestMessage to create the response
func (s *Service) CreateBlockResponse(*network.BlockRequestMessage) (*network.BlockResponseMessage, error) {
	return nil, nil
}

func (s *Service) HighestBlock() uint { return 0 }

// Start begins the chainSync and chainProcessor modules. It begins syncing in bootstrap mode
func (s *Service) Start() error {
	go s.bootstrapSyncer.Sync()
	return nil
}

// Stop stops the chainSync and chainProcessor modules
func (s *Service) Stop() error {
	return nil
}
