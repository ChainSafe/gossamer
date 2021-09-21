package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

type GetKeysRequest struct {
	Key    string
	Prefix string
	Hash   common.Hash
}

type ChildStateModule struct {
	storageAPI StorageAPI
	blockAPI   BlockAPI
}

func (cs *ChildStateModule) GetKeys(r *http.Request, req *GetKeysRequest, res *[]string) error {
	if req.Hash == common.EmptyHash {
		req.Hash = cs.blockAPI.BestBlockHash()
	}

	stateRoot, err := cs.storageAPI.GetStateRootFromBlock(&req.Hash)
	if err != nil {
		return err
	}

	trie, err := cs.storageAPI.GetStorageChild(stateRoot, []byte(req.Key))
	if err != nil {
		return err
	}

	keys := trie.GetKeysWithPrefix([]byte(req.Hash.String()))
	hexKeys := make([]string, len(keys))
	for idx, k := range keys {
		hex := common.BytesToHex(k)
		hexKeys[idx] = hex
	}

	*res = hexKeys
	return nil
}
