package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

// GetKeysRequest represents the request to retrieve the keys of a child storage
type GetKeysRequest struct {
	Key    []byte
	Prefix []byte
	Hash   common.Hash
}

// ChildStateModule is the module responsible to implement all the childstate RPC calls
type ChildStateModule struct {
	storageAPI StorageAPI
	blockAPI   BlockAPI
}

// NewChildStateModule returns a new ChildStateModule
func NewChildStateModule(s StorageAPI, b BlockAPI) *ChildStateModule {
	return &ChildStateModule{
		storageAPI: s,
		blockAPI:   b,
	}
}

// GetKeys returns the keys from the specified child storage. The keys can also be filtered based on a prefix.
func (cs *ChildStateModule) GetKeys(r *http.Request, req *GetKeysRequest, res *[]string) error {
	if req.Hash == common.EmptyHash {
		req.Hash = cs.blockAPI.BestBlockHash()
	}

	stateRoot, err := cs.storageAPI.GetStateRootFromBlock(&req.Hash)
	if err != nil {
		return err
	}

	trie, err := cs.storageAPI.GetStorageChild(stateRoot, req.Key)
	if err != nil {
		return err
	}

	keys := trie.GetKeysWithPrefix(req.Prefix)
	hexKeys := make([]string, len(keys))
	for idx, k := range keys {
		hex := common.BytesToHex(k)
		hexKeys[idx] = hex
	}

	*res = hexKeys
	return nil
}
