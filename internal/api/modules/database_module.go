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
	"github.com/ChainSafe/gossamer/common"
	"github.com/ethereum/go-ethereum/log"
)

type DatabaseModule struct {
	Database DatabaseApi
}

// P2pApi is the interface expected to implemented by `p2p` package
type DatabaseApi interface {
	GetChildKeys([]byte, []byte, common.Hash) [][]byte
	GetChildStorage([]byte, []byte, common.Hash) []byte
	GetChildStorageHash([]byte, []byte, common.Hash) common.Hash
	GetChildStorageSize([]byte, []byte, common.Hash) uint64
	GetKeys([]byte, common.Hash) [][]byte
	GetMetadata(common.Hash) []byte
	GetRuntimeVersion(common.Hash) string
	GetStorage([]byte, common.Hash) []byte
	GetStorageHash([]byte, common.Hash) common.Hash
	GetStorageSize([]byte, common.Hash) uint64
	// QueryStorage([][]byte, common.Hash, common.Hash) uint64
}

func NewDatabaseModule(databaseApi DatabaseApi) *DatabaseModule {
	return &DatabaseModule{databaseApi}
}

func (p *DatabaseModule) GetChildKeys(childStorageKey, key []byte, block common.Hash) [][]byte {
	log.Debug("[rpc] Executing Chain.getBlockHash", "params", nil)
	return p.Database.GetChildKeys(childStorageKey, key, block)
}

func (p *DatabaseModule) GetChildStorage(childStorageKey, key []byte, block common.Hash) []byte {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetChildStorage(childStorageKey, key, block)
}

func (p *DatabaseModule) GetChildStorageHash(childStorageKey, key []byte, block common.Hash) common.Hash {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetChildStorageHash(childStorageKey, key, block)
}

func (p *DatabaseModule) GetChildStorageSize(childStorageKey, key []byte, block common.Hash) uint64 {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetChildStorageSize(childStorageKey, key, block)
}

func (p *DatabaseModule) GetKeys(key []byte, block common.Hash) [][]byte {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetKeys(key, block)
}

//TODO: Figure out return type
func (p *DatabaseModule) GetMetadata(block common.Hash) []byte {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetMetadata(block)
}

func (p *DatabaseModule) GetRuntimeVersion(block common.Hash) string {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetRuntimeVersion(block)
}

func (p *DatabaseModule) GetStorage(key []byte, block common.Hash) []byte {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetStorage(key, block)
}

func (p *DatabaseModule) GetStorageHash(key []byte, block common.Hash) common.Hash {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetStorageHash(key, block)
}

func (p *DatabaseModule) GetStorageSize(key []byte, block common.Hash) uint64 {
	log.Debug("[rpc] Executing Chain.getFinalizedHead", "params", nil)
	return p.Database.GetStorageSize(key, block)
}
