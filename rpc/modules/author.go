// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package modules

import (
	"math/big"
	"net/http"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/internal/api"
)

type KeyInsertRequest struct {
	KeyType   string `json:"keyType"`
	Suri      string `json:"suri"`
	PublicKey []byte `json:"publicKey"`
}

type Extrinsic struct {
}

type ChainBlockNumberRequest *big.Int

// TODO: Waiting on Block type defined here https://github.com/ChainSafe/gossamer/pull/233
type KeyInsertResponse []byte

type KeyRotateResponse []byte

type ChainBlockHeaderResponse struct{}

type AuthorHashResponse common.Hash

// ChainModule is an RPC module providing access to storage API points.
type AuthorModule struct {
	api *api.Api
}

// NewChainModule creates a new State module.
func NewAuthorModule(api *api.Api) *AuthorModule {
	return &AuthorModule{
		api: api,
	}
}

func (cm *AuthorModule) InsertKey(r *http.Request, req *KeyInsertRequest, res *KeyInsertResponse) {
	*res = cm.api.AuthorSystem.InsertKey(req.KeyType, req.Suri, req.PublicKey)
	return
}

func (cm *AuthorModule) PendingExtrinsics(r *http.Request, req *EmptyRequest, res *ChainHashResponse) {
	*res = cm.api.AuthorSystem.PendingExtrinsics()
	return
}

func (cm *AuthorModule) removeExtrinsic(r *http.Request, req *EmptyRequest, res *ChainHashResponse) {
	return
}

func (cm *AuthorModule) rotateKeys(r *http.Request, req *EmptyRequest, res *KeyRotateResponse) {
	*res = cm.api.AuthorSystem.RotateKeys()
	return
}

// TODO: Finish implementing
// func (cm *ChainModule) submitAndWatchExtrinsic(r *http.Request, req *_, res *_) {
// 	return
// }

func (cm *AuthorModule) submitExtrinsic(r *http.Request, req *, res *AuthorHashResponse) {
	return
}
