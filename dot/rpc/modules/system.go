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
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/scale"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v2/types"
)

// SystemModule is an RPC module providing access to core API points
type SystemModule struct {
	networkAPI NetworkAPI
	systemAPI  SystemAPI
	coreAPI    CoreAPI
	storageAPI StorageAPI
	txStateAPI TransactionStateAPI
}

// EmptyRequest represents an RPC request with no fields
type EmptyRequest struct{}

// StringResponse holds the string response
type StringResponse string

// SystemHealthResponse struct to marshal json
type SystemHealthResponse common.Health

// NetworkStateString Network State represented as string so JSON encode/decoding works
type NetworkStateString struct {
	PeerID     string
	Multiaddrs []string
}

// SystemNetworkStateResponse struct to marshal json
type SystemNetworkStateResponse struct {
	NetworkState NetworkStateString `json:"networkState"`
}

// SystemPeersResponse struct to marshal json
type SystemPeersResponse []common.PeerInfo

// U64Response holds U64 response
type U64Response uint64

// StringRequest holds string request
type StringRequest struct {
	String string
}

// NewSystemModule creates a new API instance
func NewSystemModule(net NetworkAPI, sys SystemAPI, core CoreAPI,
	storage StorageAPI, txAPI TransactionStateAPI) *SystemModule {
	return &SystemModule{
		networkAPI: net, // TODO: migrate to network state
		systemAPI:  sys,
		coreAPI:    core,
		storageAPI: storage,
		txStateAPI: txAPI,
	}
}

// Chain returns the runtime chain
func (sm *SystemModule) Chain(r *http.Request, req *EmptyRequest, res *string) error {
	*res = sm.systemAPI.ChainName()
	return nil
}

// Name returns the runtime name
func (sm *SystemModule) Name(r *http.Request, req *EmptyRequest, res *string) error {
	*res = sm.systemAPI.SystemName()
	return nil
}

// ChainType returns the chain type
func (sm *SystemModule) ChainType(r *http.Request, req *EmptyRequest, res *string) error {
	*res = sm.systemAPI.ChainType()
	return nil
}

// Properties returns the runtime properties
func (sm *SystemModule) Properties(r *http.Request, req *EmptyRequest, res *interface{}) error {
	*res = sm.systemAPI.Properties()
	return nil
}

// Version returns the runtime version
func (sm *SystemModule) Version(r *http.Request, req *EmptyRequest, res *string) error {
	*res = sm.systemAPI.SystemVersion()
	return nil
}

// Health returns the information about the health of the network
func (sm *SystemModule) Health(r *http.Request, req *EmptyRequest, res *SystemHealthResponse) error {
	health := sm.networkAPI.Health()
	*res = SystemHealthResponse(health)
	return nil
}

// NetworkState returns the network state (basic information about the host)
func (sm *SystemModule) NetworkState(r *http.Request, req *EmptyRequest, res *SystemNetworkStateResponse) error {
	networkState := sm.networkAPI.NetworkState()
	res.NetworkState.PeerID = networkState.PeerID
	for _, v := range networkState.Multiaddrs {
		res.NetworkState.Multiaddrs = append(res.NetworkState.Multiaddrs, v.String())
	}
	return nil
}

// Peers returns peer information for each connected and confirmed peer
func (sm *SystemModule) Peers(r *http.Request, req *EmptyRequest, res *SystemPeersResponse) error {
	peers := sm.networkAPI.Peers()
	*res = peers
	return nil
}

// NodeRoles Returns the roles the node is running as.
func (sm *SystemModule) NodeRoles(r *http.Request, req *EmptyRequest, res *[]interface{}) error {
	resultArray := []interface{}{}

	role := sm.networkAPI.NodeRoles()
	switch role {
	case 1:
		resultArray = append(resultArray, "Full")
	case 2:
		resultArray = append(resultArray, "LightClient")
	case 4:
		resultArray = append(resultArray, "Authority")
	default:
		resultArray = append(resultArray, "UnknownRole")
		uknrole := []interface{}{}
		uknrole = append(uknrole, role)
		resultArray = append(resultArray, uknrole)
	}

	*res = resultArray
	return nil
}

// AccountNextIndex Returns the next valid index (aka. nonce) for given account.
func (sm *SystemModule) AccountNextIndex(r *http.Request, req *StringRequest, res *U64Response) error {
	if req == nil || len(req.String) == 0 {
		return errors.New("account address must be valid")
	}
	addressPubKey := crypto.PublicAddressToByteArray(common.Address(req.String))

	// check pending transactions for extrinsics singed by addressPubKey
	pending := sm.txStateAPI.Pending()
	nonce := uint64(0)
	found := false
	for _, v := range pending {
		var ext ctypes.Extrinsic
		err := ctypes.DecodeFromBytes(v.Extrinsic[1:], &ext)
		if err != nil {
			return err
		}
		extSigner, err := common.HexToBytes(fmt.Sprintf("0x%x", ext.Signature.Signer.AsAccountID))
		if err != nil {
			return err
		}
		if bytes.Equal(extSigner, addressPubKey) {
			found = true
			sigNonce := big.Int(ext.Signature.Nonce)
			if sigNonce.Uint64() > nonce {
				nonce = sigNonce.Uint64()
			}
		}
	}

	if found {
		*res = U64Response(nonce)
		return nil
	}

	// no extrinsic signed by request found in pending transactions, so look in storage
	// get metadata to build storage storageKey
	rawMeta, err := sm.coreAPI.GetMetadata(nil)
	if err != nil {
		return err
	}
	sdMeta, err := scale.Decode(rawMeta, []byte{})
	if err != nil {
		return err
	}
	var metadata ctypes.Metadata
	err = ctypes.DecodeFromBytes(sdMeta.([]byte), &metadata)
	if err != nil {
		return err
	}

	storageKey, err := ctypes.CreateStorageKey(&metadata, "System", "Account", addressPubKey, nil)
	if err != nil {
		return err
	}

	accountRaw, err := sm.storageAPI.GetStorage(nil, storageKey)
	if err != nil {
		return err
	}

	var accountInfo ctypes.AccountInfo
	err = ctypes.DecodeFromBytes(accountRaw, &accountInfo)
	if err != nil {
		return err
	}

	*res = U64Response(accountInfo.Nonce)
	return nil
}
