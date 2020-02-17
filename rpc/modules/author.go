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
)

type AuthorModule struct {
	coreApi CoreApi
}

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

// KeyInsertResponse []byte
// TODO: Waiting on Block type defined here https://github.com/ChainSafe/gossamer/pull/233
type KeyInsertResponse []byte

type PendingExtrinsicsResponse [][]byte

type RemoveExtrinsicsResponse []common.Hash

type KeyRotateResponse []byte

type ExtrinsicStatus struct {
	IsFuture    bool
	IsReady     bool
	IsFinalized bool
	AsFinalized common.Hash
	IsUsurped   bool
	AsUsurped   common.Hash
	IsBroadcast bool
	AsBroadcast []string
	IsDropped   bool
	IsInvalid   bool
}

type ExtrinsicHashResponse common.Hash

// NewAuthorModule creates a new Author module.
func NewAuthorModule(api CoreApi) *AuthorModule {
	return &AuthorModule{
		coreApi: api,
	}
}

// InsertKey snserts a key into the keystore
func (cm *AuthorModule) InsertKey(r *http.Request, req *KeyInsertRequest, res *KeyInsertResponse) {
	_ = cm.coreApi
}

// PendingExtrinsics returns all pending extrinsics
func (cm *AuthorModule) PendingExtrinsics(r *http.Request, req *EmptyRequest, res *PendingExtrinsicsResponse) {
}

// RemoveExtrinsic removes given extrinsic from the pool and temporarily ban it to prevent reimporting
func (cm *AuthorModule) RemoveExtrinsic(r *http.Request, req *ExtrinsicOrHashRequest, res *RemoveExtrinsicsResponse) {
}

// RotateKeys generates new session keys and returns the corresponding public keys
func (cm *AuthorModule) RotateKeys(r *http.Request, req *EmptyRequest, res *KeyRotateResponse) {
}

// SubmitAndWatchExtrinsic submits an extrinsic and watches it
func (cm *AuthorModule) SubmitAndWatchExtrinsic(r *http.Request, req *Extrinsic, res *ExtrinsicStatus) {
}

// SubmitExtrinsic submits a fully formatted extrinsic for block inclusion
func (cm *AuthorModule) SubmitExtrinsic(r *http.Request, req *Extrinsic, res *ExtrinsicHashResponse) {
}
