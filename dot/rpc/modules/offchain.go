package modules

import (
	"fmt"
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

const (
	offchainPersistent = "PERSISTENT"
	offchainLocal      = "LOCAL"
)

// OffchainLocalStorageGet represents the request format to retrieve data from offchain storage
type OffchainLocalStorageGet struct {
	Kind string
	Key  string
}

// OffchainLocalStorageSet represents the request format to store data into offchain storage
type OffchainLocalStorageSet struct {
	Kind  string
	Key   string
	Value string
}

// OffchainModule defines the RPC module to Offchain methods
type OffchainModule struct {
	nodeStorage RuntimeStorageAPI
}

// NewOffchainModule creates a RPC module to Offchain methods
func NewOffchainModule(ns RuntimeStorageAPI) *OffchainModule {
	return &OffchainModule{
		nodeStorage: ns,
	}
}

// LocalStorageGet get offchain local storage under given key and prefix
func (s *OffchainModule) LocalStorageGet(_ *http.Request, req *OffchainLocalStorageGet, res *StringResponse) error {
	var (
		v   []byte
		key []byte
		err error
	)

	if key, err = common.HexToBytes(req.Key); err != nil {
		return err
	}

	switch req.Kind {
	case offchainPersistent:
		v, err = s.nodeStorage.GetPersistent(key)
	case offchainLocal:
		v, err = s.nodeStorage.GetLocal(key)
	default:
		return fmt.Errorf("storage kind not found: %s", req.Kind)
	}

	if err != nil {
		return err
	}

	*res = StringResponse(common.BytesToHex(v))
	return nil
}

// LocalStorageSet set offchain local storage under given key and prefix
func (s *OffchainModule) LocalStorageSet(_ *http.Request, req *OffchainLocalStorageSet, _ *StringResponse) error {
	var (
		val []byte
		key []byte
		err error
	)

	if key, err = common.HexToBytes(req.Key); err != nil {
		return err
	}

	if val, err = common.HexToBytes(req.Value); err != nil {
		return err
	}

	switch req.Kind {
	case offchainPersistent:
		err = s.nodeStorage.SetPersistent(key, val)
	case offchainLocal:
		err = s.nodeStorage.SetLocal(key, val)
	default:
		return fmt.Errorf("storage kind not found: %s", req.Kind)
	}

	if err != nil {
		return err
	}

	return nil
}
