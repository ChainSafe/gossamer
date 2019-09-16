// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.

// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.

// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package module

import (
	"github.com/ethereum/go-ethereum/log"
	"githum.com/ChainSafe/gossamer/common"
	core "githum.com/ChainSafe/gossamer/core"
)

type AuthorModule struct {
	Author AuthorApi
}

// AuthorApi is the interface expected to implemented by `p2p` package
type AuthorApi interface {
	InsertKey(keyType, suri string, publicKey []byte) []byte
	PendingExtrinsics() []core.BlockBody
	// RemoveExtrinsic(bytesOrHash []core.BlockBody) []common.Hash
	RotateKeys() []byte
	SubmitExtrinsic(core.BlockBody) common.Hash
}

func NewAuthorModule(authorapi AuthorApi) *AuthorModule {
	return &AuthorModule{authorapi}
}

func (a *AuthorModule) InsertKey(keyType, suri string, publicKey []byte) []byte {
	log.Debug("[rpc] Executing Author.InsertKey", "params", nil)
	return a.Author.InsertKey(keyType, suri, publicKey)
}

func (a *AuthorModule) PendingExtrinsics() []core.BlockBody {
	log.Debug("[rpc] Executing Author.PendingExtrinsics", "params", nil)
	return a.Author.PendingExtrinsics()
}

// func (a *AuthorModule) RemoveExtrinsic(extrinsic core.BlockBody) common.Hash {

// }

func (a *AuthorModule) RotateKeys() []byte {
	log.Debug("[rpc] Executing Author.RotateKeys", "params", nil)
	return a.Author.RotateKeys()
}

func (a *AuthorModule) SubmitExtrinsic(extrinsic core.BlockBody) common.Hash {
	log.Debug("[rpc] Executing Author.SubmitExtrinsic", "params", nil)
	return a.Author.SubmitExtrinsic(extrinsic)
}
