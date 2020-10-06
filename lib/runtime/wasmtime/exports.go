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

package wasmtime

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/types"
	gssmrruntime "github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/ChainSafe/gossamer/lib/transaction"
)

func (in *Instance) Version() (*gssmrruntime.VersionAPI, error) {
	res, err := in.exec(gssmrruntime.CoreVersion, []byte{})
	if err != nil {
		return nil, err
	}

	version := &gssmrruntime.VersionAPI{
		RuntimeVersion: &gssmrruntime.Version{},
		API:            nil,
	}

	err = version.Decode(res)
	if err != nil {
		return nil, err
	}

	return version, nil
}

func (in *Instance) BabeConfiguration() (*types.BabeConfiguration, error) {
	ret, err := in.exec(gssmrruntime.BabeAPIConfiguration, []byte{})
	if err != nil {
		return nil, err
	}

	cfg, err := scale.Decode(ret, new(types.BabeConfiguration))
	if err != nil {
		return nil, err
	}

	return cfg.(*types.BabeConfiguration), nil
}

func (in *Instance) GrandpaAuthorities() ([]*types.Authority, error) {
	return nil, nil
}

func (in *Instance) ValidateTransaction(e types.Extrinsic) (*transaction.Validity, error) {
	return nil, nil
}

func (in *Instance) InitializeBlock(header *types.Header) error {
	encodedHeader, err := scale.Encode(header)
	if err != nil {
		return fmt.Errorf("cannot encode header: %s", err)
	}

	_, err = in.exec(gssmrruntime.CoreInitializeBlock, encodedHeader)
	return err
}

func (in *Instance) InherentExtrinsics(data []byte) ([]byte, error) {
	return nil, nil
}

func (in *Instance) ApplyExtrinsic(data types.Extrinsic) ([]byte, error) {
	return nil, nil
}

func (in *Instance) FinalizeBlock() (*types.Header, error) {
	return nil, nil
}

func (in *Instance) ExecuteBlock(block *types.Block) ([]byte, error) {
	return nil, nil
}
