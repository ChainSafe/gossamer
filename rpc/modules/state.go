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

type StateCallRequest struct {
	Method string      `json:"method"`
	Data   []byte      `json:"data"`
	Block  common.Hash `json:"block"`
}

type StateChildStorageRequest struct {
	ChildStorageKey []byte      `json:"childStorageKey"`
	Key             []byte      `json:"key"`
	Block           common.Hash `json:"block"`
}

type StateStorageQueryRequest struct {
	Key   []byte      `json:"key"`
	Block common.Hash `json:"block"`
}

type StateBlockHashQuery common.Hash

type StateStorageQueryRangeRequest struct {
	Keys       []byte      `json:"keys"`
	StartBlock common.Hash `json:"startBlock"`
	EndBlock   common.Hash `json:"block"`
}

type StateCallResponse []byte

type StateKeysResponse [][]byte

type StateStorageDataResponse []byte

type StateStorageHashResponse common.Hash

type StateStorageSizeResponse uint64

type StateStorageKeysResponse [][]byte

// TODO: Determine actual type
type StateMetadataResponse []byte

type StateRuntimeVersionResponse string

// StateModule is an RPC module providing access to storage API points.
type StateModule struct {
	api *api.Api
}

// NewStateModule creates a new State module.
func NewStateModule(api *api.Api) *SystemModule {
	return &SystemModule{
		api: api,
	}
}

func (sm *StateModule) Call(r *http.Request, req *StateCallRequest, res *StateCallResponse) {
	return
}

func (sm *StateModule) GetChildKeys(r *http.Request, req *StateChildStorageRequest, res *StateKeysResponse) {
	return
}

func (sm *StateModule) GetChildStorage(r *http.Request, req *StateChildStorageRequest, res *StateStorageDataResponse) {
	return
}

func (sm *StateModule) GetChildStorageHash(r *http.Request, req *StateChildStorageRequest, res *StateStorageHashResponse) {
	return
}

func (sm *StateModule) GetChildStorageSize(r *http.Request, req *StateChildStorageRequest, res *StateStorageSizeResponse) {
	return
}

func (sm *StateModule) GetKeys(r *http.Request, req *StateChildStorageRequest, res *StateStorageKeysResponse) {
	return
}

func (sm *StateModule) GetMetadata(r *http.Request, req *StateStorageQueryRequest, res *StateMetadataResponse) {
	return
}

func (sm *StateModule) GetRuntimeVersion(r *http.Request, req *StateBlockHashQuery, res *StateRuntimeVersionResponse) {
	return
}

func (sm *StateModule) GetStorage(r *http.Request, req *StateStorageQueryRequest, res *StateStorageDataResponse) {
	return
}

func (sm *StateModule) GetStorageHash(r *http.Request, req *StateStorageQueryRequest, res *StateStorageHashResponse) {
	return
}

func (sm *StateModule) GetStorageSize(r *http.Request, req *StateStorageQueryRequest, res *StateStorageSizeResponse) {
	return
}

// TDDO: Complete implementation
//func (sm *StateModule) QueryStorage(r *http.Request, _ _, _ _) {
//
//}
//
//func (sm *StateModule) SubscribeRuntimeVersion(r *http.Request, req *_, res *_) {
//
//}
//
//func (sm *StateModule) SubscribeStorage(r *http.Request, req *_, res *_) {
//
//}
