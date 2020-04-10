package modules

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/ChainSafe/gossamer/dot/core"
	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
	"github.com/stretchr/testify/require"

	"github.com/ChainSafe/gossamer/dot/core/types"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

var testExt = []byte{3, 16, 110, 111, 111, 116, 1, 64, 103, 111, 115, 115, 97, 109, 101, 114, 95, 105, 115, 95, 99, 111, 111, 108}

// testGenesisHeader is a test block header
var testGenesisHeader = &types.Header{
	Number:    big.NewInt(0),
	StateRoot: trie.EmptyHash,
}

// newTestService creates a new test core service
func newTestService(t *testing.T, cfg *core.Config) *core.Service {
	if cfg == nil {
		rt := runtime.NewTestRuntime(t, runtime.POLKADOT_RUNTIME_c768a7e4c70e)
		cfg = &core.Config{
			Runtime:         rt,
			IsBabeAuthority: false,
		}
	}

	if cfg.Keystore == nil {
		cfg.Keystore = keystore.NewKeystore()
	}

	if cfg.NewBlocks == nil {
		cfg.NewBlocks = make(chan types.Block)
	}

	if cfg.MsgRec == nil {
		cfg.MsgRec = make(chan network.Message, 10)
	}

	if cfg.MsgSend == nil {
		cfg.MsgSend = make(chan network.Message, 10)
	}

	if cfg.SyncChan == nil {
		cfg.SyncChan = make(chan *big.Int, 10)
	}

	stateSrvc := state.NewService("")
	stateSrvc.UseMemDB()

	genesisData := new(genesis.Data)

	err := stateSrvc.Initialize(genesisData, testGenesisHeader, trie.NewEmptyTrie())
	require.Nil(t, err)

	err = stateSrvc.Start()
	require.Nil(t, err)

	if cfg.BlockState == nil {
		cfg.BlockState = stateSrvc.Block
	}

	if cfg.StorageState == nil {
		cfg.StorageState = stateSrvc.Storage
	}

	s, err := core.NewService(cfg)
	require.Nil(t, err)

	return s
}

func TestAuthorModule_Pending(t *testing.T) {
	txQueue := state.NewTransactionQueue()
	auth := NewAuthorModule(nil, txQueue)

	res := new(PendingExtrinsicsResponse)
	err := auth.PendingExtrinsics(nil, nil, res)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*res, PendingExtrinsicsResponse([][]byte{})) {
		t.Errorf("Fail: expected: %+v got: %+v\n", res, &[][]byte{})
	}

	vtx := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(testExt),
		Validity:  new(transaction.Validity),
	}

	txQueue.Push(vtx)

	err = auth.PendingExtrinsics(nil, nil, res)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := vtx.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*res, PendingExtrinsicsResponse([][]byte{expected})) {
		t.Errorf("Fail: expected: %+v got: %+v\n", res, &[][]byte{expected})
	}
}

func TestAuthorModule_SubmitExtrinsic(t *testing.T) {
	txQueue := state.NewTransactionQueue()
	auth := NewAuthorModule(nil, txQueue)
	ext := Extrinsic(fmt.Sprintf("0x%x", testExt))

	res := new(ExtrinsicHashResponse)

	err := auth.SubmitExtrinsic(nil, &ext, res)
	if err != nil {
		t.Fatal(err)
	}

	expected := &transaction.ValidTransaction{
		Extrinsic: types.NewExtrinsic(testExt),
		Validity:  nil,
	}

	inQueue := txQueue.Pop()
	if !reflect.DeepEqual(inQueue, expected) {
		t.Fatalf("Fail: got %v expected %v", inQueue, expected)
	}
}

func TestAuthorModule_InsertKey_Valid(t *testing.T) {
	cs := newTestService(t, nil)

	auth := NewAuthorModule(cs, nil)
	req := &KeyInsertRequest{"babe", "0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309", "0x6246ddf254e0b4b4e7dffefc8adf69d212b98ac2b579c362b473fec8c40b4c0a"}
	res := &KeyInsertResponse{}
	err := auth.InsertKey(nil, req, res)
	require.Nil(t, err)

	require.Len(t, *res, 0) // zero len result on success
}

func TestAuthorModule_InsertKey_InValid(t *testing.T) {
	cs := newTestService(t, nil)

	auth := NewAuthorModule(cs, nil)
	req := &KeyInsertRequest{"babe", "0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309", "0x0000000000000000000000000000000000000000000000000000000000000000"}
	res := &KeyInsertResponse{}
	err := auth.InsertKey(nil, req, res)
	require.EqualError(t, err, "generated public key does not equal provide public key")
}

func TestAuthorModule_InsertKey_UnknownKeyType(t *testing.T) {
	cs := newTestService(t, nil)

	auth := NewAuthorModule(cs, nil)
	req := &KeyInsertRequest{"mack", "0xb7e9185065667390d2ad952a5324e8c365c9bf503dcf97c67a5ce861afe97309", "0x6246ddf254e0b4b4e7dffefc8adf69d212b98ac2b579c362b473fec8c40b4c0a"}
	res := &KeyInsertResponse{}
	err := auth.InsertKey(nil, req, res)
	require.EqualError(t, err, "cannot decode key: invalid key type")

}
