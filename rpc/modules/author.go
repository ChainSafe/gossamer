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
	"net/http"

	"github.com/ChainSafe/gossamer/common"
	"github.com/ChainSafe/gossamer/internal/api"
)

type KeyInsertRequest struct {
	KeyType   string `json:"keyType"`
	Suri      string `json:"suri"`
	PublicKey []byte `json:"publicKey"`
}

type Extrinsic []byte

type ExtrinsicOrHash struct {
	Hash      common.Hash
	Extrinsic []byte
}
type ExtrinsicOrHashRequest []ExtrinsicOrHash

// TODO: Waiting on Block type defined here https://github.com/ChainSafe/gossamer/pull/233
type KeyInsertResponse []byte

type PendingExtrinsicsResponse [][]byte

type RemoveExtrinsicsResponse []common.Hash

type KeyRotateResponse []byte

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

// Insert a key into the keystore
func (cm *AuthorModule) InsertKey(r *http.Request, req *KeyInsertRequest, res *KeyInsertResponse) {
	*res = cm.api.InsertKey(req.KeyType, req.Suri, req.PublicKey)
	return
}

// Returns all pending extrinsics
func (cm *AuthorModule) PendingExtrinsics(r *http.Request, req *EmptyRequest, res *PendingExtrinsicsResponse) {
	*res = cm.api.PendingExtrinsics()
	return
}

// Remove given extrinsic from the pool and temporarily ban it to prevent reimporting
func (cm *AuthorModule) RemoveExtrinsic(r *http.Request, req *ExtrinsicOrHashRequest, res *RemoveExtrinsicsResponse) {
	*res = cm.api.RemoveExtrinsics(*req)
	return
}

// Generate new session keys and returns the corresponding public keys
func (cm *AuthorModule) RotateKeys(r *http.Request, req *EmptyRequest, res *KeyRotateResponse) {
	*res = cm.api.RotateKeys()
	return
}

// Submit a fully formatted extrinsic for block inclusion
func (cm *AuthorModule) SubmitExtrinsic(r *http.Request, req *Extrinsic, res *AuthorHashResponse) {
	*res = cm.api.SubmitExtrinsic(*req)
	return
}
