// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"bytes"
	"errors"
	"math/big"
	"net/http"
	"strings"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/btcsuite/btcutil/base58"
	ctypes "github.com/centrifuge/go-substrate-rpc-client/v3/types"
)

// SystemModule is an RPC module providing access to core API points
type SystemModule struct {
	networkAPI NetworkAPI
	systemAPI  SystemAPI
	coreAPI    CoreAPI
	storageAPI StorageAPI
	txStateAPI TransactionStateAPI
	blockAPI   BlockAPI
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

// SyncStateResponse is the struct to return on the system_syncState rpc call
type SyncStateResponse struct {
	CurrentBlock  uint32 `json:"currentBlock"`
	HighestBlock  uint32 `json:"highestBlock"`
	StartingBlock uint32 `json:"startingBlock"`
}

// NewSystemModule creates a new API instance
func NewSystemModule(net NetworkAPI, sys SystemAPI, core CoreAPI,
	storage StorageAPI, txAPI TransactionStateAPI, blockAPI BlockAPI) *SystemModule {
	return &SystemModule{
		networkAPI: net,
		systemAPI:  sys,
		coreAPI:    core,
		storageAPI: storage,
		txStateAPI: txAPI,
		blockAPI:   blockAPI,
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
	if req == nil || req.String == "" {
		return errors.New("account address must be valid")
	}
	addressPubKey := crypto.PublicAddressToByteArray(common.Address(req.String))

	// check pending transactions for extrinsics singed by addressPubKey
	pending := sm.txStateAPI.Pending()
	nonce := uint64(0)
	found := false
	for _, v := range pending {
		var ext ctypes.Extrinsic
		err := ctypes.DecodeFromBytes(v.Extrinsic, &ext)
		if err != nil {
			return err
		}

		extSigner := [32]byte(ext.Signature.Signer.AsID)
		if bytes.Equal(extSigner[:], addressPubKey) {
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
	var sdMeta []byte
	err = scale.Unmarshal(rawMeta, &sdMeta)
	if err != nil {
		return err
	}
	var metadata ctypes.Metadata
	err = ctypes.DecodeFromBytes(sdMeta, &metadata)
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

// SyncState Returns the state of the syncing of the node.
func (sm *SystemModule) SyncState(r *http.Request, req *EmptyRequest, res *SyncStateResponse) error {
	h, err := sm.blockAPI.GetHeader(sm.blockAPI.BestBlockHash())
	if err != nil {
		return err
	}

	*res = SyncStateResponse{
		CurrentBlock:  uint32(h.Number.Int64()),
		HighestBlock:  uint32(sm.networkAPI.HighestBlock()),
		StartingBlock: uint32(sm.networkAPI.StartingBlock()),
	}
	return nil
}

// LocalListenAddresses Returns the libp2p multiaddresses that the local node is listening on
func (sm *SystemModule) LocalListenAddresses(r *http.Request, req *EmptyRequest, res *[]string) error {
	netstate := sm.networkAPI.NetworkState()

	if len(netstate.Multiaddrs) < 1 {
		return errors.New("multiaddress list is empty")
	}

	addrs := make([]string, len(netstate.Multiaddrs))

	for i, ma := range netstate.Multiaddrs {
		addrs[i] = ma.String()
	}

	*res = addrs
	return nil
}

// LocalPeerId Returns the base58-encoded PeerId fo the node.
func (sm *SystemModule) LocalPeerId(r *http.Request, req *EmptyRequest, res *string) error { //nolint
	netstate := sm.networkAPI.NetworkState()
	if netstate.PeerID == "" {
		return errors.New("peer id cannot be empty")
	}

	*res = base58.Encode([]byte(netstate.PeerID))
	return nil
}

// AddReservedPeer adds a reserved peer. The string parameter should encode a p2p multiaddr.
func (sm *SystemModule) AddReservedPeer(r *http.Request, req *StringRequest, res *[]byte) error {
	if strings.TrimSpace(req.String) == "" {
		return errors.New("cannot add an empty reserved peer")
	}

	return sm.networkAPI.AddReservedPeers(req.String)
}

// RemoveReservedPeer remove a reserved peer. The string should encode only the PeerId
func (sm *SystemModule) RemoveReservedPeer(r *http.Request, req *StringRequest, res *[]byte) error {
	if strings.TrimSpace(req.String) == "" {
		return errors.New("cannot remove an empty reserved peer")
	}

	return sm.networkAPI.RemoveReservedPeers(req.String)
}
