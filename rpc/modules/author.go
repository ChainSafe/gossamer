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
	"github.com/ChainSafe/gossamer/core"
	"github.com/ChainSafe/gossamer/internal/api"
)

type KeyInsertRequest struct {
	KeyType   string `json:"keyType"`
	Suri      string `json:"suri"`
	PublicKey []byte `json:"publicKey"`
}

type Extrinsic struct {
}

type ExtrinsicOrHashRequest struct {
	Extrinsic core.Block
	Hash      common.Hash
}

type SubmitExtrinsicRequest struct {
	Extrinsic core.Block
}

type ChainBlockNumberRequest *big.Int

// TODO: Waiting on Block type defined here https://github.com/ChainSafe/gossamer/pull/233
type KeyInsertResponse []byte

type KeyRotateResponse []byte

type ChainBlockHeaderResponse struct{}

type AuthorHashResponse common.Hash

// ChainModule is an RPC module providing access to storage API points.
type AuthorRPC struct {
	api *api.Api
}

// NewChainModule creates a new State module.
func NewAuthorRPC(api *api.Api) *AuthorRPC {
	return &AuthorRPC{
		api: api,
	}
}

func (cm *AuthorRPC) InsertKey(r *http.Request, req *KeyInsertRequest, res *KeyInsertResponse) {
}

func (cm *AuthorRPC) PendingExtrinsics(r *http.Request, req *EmptyRequest, res *ChainHashResponse) {
}

func (cm *AuthorRPC) RemoveExtrinsic(r *http.Request, req *ExtrinsicOrHashRequest, res *ChainHashResponse) {
}

func (cm *AuthorRPC) RotateKeys(r *http.Request, req *EmptyRequest, res *KeyRotateResponse) {
}

// TODO: Finish implementing
// func (cm *ChainModule) submitAndWatchExtrinsic(r *http.Request, req *_, res *_) {
// 	return
// }

func (cm *AuthorRPC) SubmitExtrinsic(r *http.Request, req *SubmitExtrinsicRequest, res *AuthorHashResponse) {
}
