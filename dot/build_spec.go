// Copyright 2020 ChainSafe Systems (ON) Corp.
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
package dot

import (
	"encoding/json"
	"fmt"
	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/ChainSafe/gossamer/lib/scale"
	log "github.com/ChainSafe/log15"
)

type BuildSpec struct {
	genesis *genesis.Genesis
}

func MakeBuildSpec() *BuildSpec {
	return &BuildSpec{}
}

func (b *BuildSpec) ToJSON() ([]byte, error) {
	tmpGen := &genesis.Genesis{
		Name:       b.genesis.Name,
		ID:         b.genesis.ID,
		Bootnodes:  b.genesis.Bootnodes,
		ProtocolID: b.genesis.ProtocolID,
		Genesis:    genesis.Fields{
			Runtime: b.genesis.GenesisFields().Runtime,
		},
	}
	return json.MarshalIndent(tmpGen, "", "    ")
}

func (b *BuildSpec) ToJSONRaw() ([]byte, error) {
	tmpGen := &genesis.Genesis{
		Name:       b.genesis.Name,
		ID:         b.genesis.ID,
		Bootnodes:  b.genesis.Bootnodes,
		ProtocolID: b.genesis.ProtocolID,
		Genesis:    genesis.Fields{
			Raw: b.genesis.GenesisFields().Raw,
		},
	}
	return json.MarshalIndent(tmpGen, "", "    ")
}
func BuildFromGenesis(path string) (*BuildSpec, error) {
	gen, err := genesis.NewGenesisFromJSON(path)
	if err != nil {
		return nil, err
	}
	bs := &BuildSpec{
		genesis: gen,
	}
	return bs, nil
}

func BuildFromDB(path string) (*BuildSpec, error) {
	tmpGen := &genesis.Genesis{
		Name:       "",
		ID:         "",
		Bootnodes:  nil,
		ProtocolID: "",
		Genesis:    genesis.Fields{
			Runtime: nil,
		},
	}
	tmpGen.Genesis.Raw[0] = make(map[string]string)
	tmpGen.Genesis.Runtime = make(map[string]map[string]interface{})

	fmt.Printf("%v\n", tmpGen)

	stateSrvc := state.NewService(path, log.LvlDebug)

	// start state service (initialize state database)
	err := stateSrvc.Start()
	if err != nil {
		return nil, err
	}


	ent := stateSrvc.Storage.Entries()
	genesis.BuildFromMap(ent, tmpGen)

	fmt.Printf("built %v\n", tmpGen.Genesis.Raw[0]["3a636f6465"][:100])
	codStr := fmt.Sprintf("%v", tmpGen.Genesis.Runtime["system"]["code"])
	fmt.Printf("built runtime %v\n", codStr[:100])

	fmt.Printf("genesesis Data Key %x\n", common.GenesisDataKey)
	// set genesisData
	gd, err := stateSrvc.DB().Get(common.GenesisDataKey)
	fmt.Printf("gd %s\n", gd)

	gData, err := scale.Decode(gd, &genesis.Data{})
	if err != nil {
		return nil, err
	}
	tmpGen.Name = gData.(*genesis.Data).Name
	tmpGen.ID = gData.(*genesis.Data).ID
	// todo bootnodes
	//tmpGen.Bootnodes = gData.(*genesis.Data).Bootnodes
	tmpGen.ProtocolID = gData.(*genesis.Data).ProtocolID
	
	bs := &BuildSpec{
		genesis: tmpGen,
	}
	return bs, nil
}

