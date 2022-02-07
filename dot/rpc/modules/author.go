// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package modules

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var ErrProvidedKeyDoesNotMatch = errors.New("generated public key does not equal provide public key")

// AuthorModule holds a pointer to the API
type AuthorModule struct {
	logger     log.LeveledLogger
	coreAPI    CoreAPI
	txStateAPI TransactionStateAPI
}

// HasSessionKeyRequest is used to receive the rpc data
type HasSessionKeyRequest struct {
	PublicKeys string
}

// KeyInsertRequest is used as model for the JSON
type KeyInsertRequest struct {
	Type      string
	Seed      string
	PublicKey string
}

// Extrinsic represents a hex-encoded extrinsic
type Extrinsic struct {
	Data string
}

// ExtrinsicOrHash is a type for Hash and Extrinsic array of bytes
type ExtrinsicOrHash struct {
	Hash      common.Hash
	Extrinsic []byte
}

// ExtrinsicOrHashRequest is a array of ExtrinsicOrHash
type ExtrinsicOrHashRequest []ExtrinsicOrHash

// KeyInsertResponse []byte
type KeyInsertResponse []byte

// PendingExtrinsicsResponse is a bi-dimensional array of bytes for allocating the pending extrinsics
type PendingExtrinsicsResponse []string

// RemoveExtrinsicsResponse is a array of hash used to Remove extrinsics
type RemoveExtrinsicsResponse []common.Hash

// KeyRotateResponse is a byte array used to rotate
type KeyRotateResponse []byte

// HasSessionKeyResponse is the response to the RPC call author_hasSessionKeys
type HasSessionKeyResponse bool

// KeyTypeID represents the key type of a session key
type keyTypeID [4]uint8

// DecodedKey is the representation of a scaled decoded public key
type decodedKey struct {
	Data []uint8
	Type keyTypeID
}

// ExtrinsicStatus holds the actual valid statuses
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

// ExtrinsicHashResponse is used as Extrinsic hash response
type ExtrinsicHashResponse string

// NewAuthorModule creates a new Author module.
func NewAuthorModule(logger *log.Logger, coreAPI CoreAPI, txStateAPI TransactionStateAPI) *AuthorModule {
	logger = logger.New(log.AddContext("service", "RPC"), log.AddContext("module", "author"))
	return &AuthorModule{
		logger:     logger,
		coreAPI:    coreAPI,
		txStateAPI: txStateAPI,
	}
}

// HasSessionKeys checks if the keystore has private keys for the given session public keys.
func (am *AuthorModule) HasSessionKeys(r *http.Request, req *HasSessionKeyRequest, res *HasSessionKeyResponse) error {
	pubKeysBytes, err := common.HexToBytes(req.PublicKeys)
	if err != nil {
		return err
	}

	pkeys, err := scale.Marshal(pubKeysBytes)
	if err != nil {
		return err
	}

	data, err := am.coreAPI.DecodeSessionKeys(pkeys)
	if err != nil {
		*res = false
		return err
	}

	var decodedKeys *[]decodedKey
	err = scale.Unmarshal(data, &decodedKeys)
	if err != nil {
		return err
	}

	if decodedKeys == nil || len(*decodedKeys) < 1 {
		*res = false
		return nil
	}

	for _, key := range *decodedKeys {
		encType := keystore.Name(key.Type[:])
		ok, err := am.coreAPI.HasKey(common.BytesToHex(key.Data), string(encType))

		if err != nil || !ok {
			*res = false
			return err
		}
	}

	*res = true
	return nil
}

// InsertKey inserts a key into the keystore
func (am *AuthorModule) InsertKey(r *http.Request, req *KeyInsertRequest, _ *KeyInsertResponse) error {
	keyReq := *req

	keyBytes, err := common.HexToBytes(req.Seed)
	if err != nil {
		return err
	}

	keyPair, err := keystore.DecodeKeyPairFromHex(keyBytes, keystore.DetermineKeyType(keyReq.Type))
	if err != nil {
		return err
	}

	//strings.EqualFold compare using case-insensitivity.
	if !strings.EqualFold(keyPair.Public().Hex(), keyReq.PublicKey) {
		return ErrProvidedKeyDoesNotMatch
	}

	err = am.coreAPI.InsertKey(keyPair, keyReq.Type)
	if err != nil {
		return err
	}

	am.logger.Info("inserted key " + keyPair.Public().Hex() + " into keystore")
	return nil
}

// HasKey Checks if the keystore has private keys for the given public key and key type.
func (am *AuthorModule) HasKey(r *http.Request, req *[]string, res *bool) error {
	reqKey := *req
	var err error
	*res, err = am.coreAPI.HasKey(reqKey[0], reqKey[1])
	return err
}

// PendingExtrinsics Returns all pending extrinsics
func (am *AuthorModule) PendingExtrinsics(r *http.Request, req *EmptyRequest, res *PendingExtrinsicsResponse) error {
	pending := am.txStateAPI.Pending()
	resp := make([]string, len(pending))
	for idx, tx := range pending {
		resp[idx] = common.BytesToHex(tx.Extrinsic)
	}

	*res = PendingExtrinsicsResponse(resp)
	return nil
}

// RemoveExtrinsic Remove given extrinsic from the pool and temporarily ban it to prevent reimporting
func (am *AuthorModule) RemoveExtrinsic(r *http.Request, _ *ExtrinsicOrHashRequest, _ *RemoveExtrinsicsResponse) error {
	return nil
}

// RotateKeys Generate new session keys and returns the corresponding public keys
func (am *AuthorModule) RotateKeys(r *http.Request, req *EmptyRequest, res *KeyRotateResponse) error {
	return nil
}

// SubmitAndWatchExtrinsic Submit and subscribe to watch an extrinsic until unsubscribed
func (am *AuthorModule) SubmitAndWatchExtrinsic(r *http.Request, req *Extrinsic, res *ExtrinsicStatus) error {
	return nil
}

// SubmitExtrinsic Submit a fully formatted extrinsic for block inclusion
func (am *AuthorModule) SubmitExtrinsic(r *http.Request, req *Extrinsic, res *ExtrinsicHashResponse) error {
	extBytes, err := common.HexToBytes(req.Data)
	if err != nil {
		return err
	}
	ext := types.Extrinsic(extBytes)

	*res = ExtrinsicHashResponse(ext.Hash().String())
	err = am.coreAPI.HandleSubmittedExtrinsic(ext)
	return err
}
