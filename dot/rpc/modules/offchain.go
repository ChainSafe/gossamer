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

type OffchainLocalStorageGet struct {
	Kind string
	Key  string
}

type OffchainLocalStorageSet struct {
	Kind  string
	Key   string
	Value string
}

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
func (s *OffchainModule) LocalStorageGet(r *http.Request, req *OffchainLocalStorageGet, res *StringResponse) error {
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

func (s *OffchainModule) LocalStorageSet(r *http.Request, req *OffchainLocalStorageSet, res *StringResponse) error {
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
