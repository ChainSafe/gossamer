// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"github.com/ChainSafe/gossamer/lib/common"
)

// Genesis stores the data parsed from the genesis configuration file
type Genesis struct {
	Name               string                 `json:"name"`
	ID                 string                 `json:"id"`
	ChainType          string                 `json:"chainType"`
	Bootnodes          []string               `json:"bootNodes"`
	TelemetryEndpoints []interface{}          `json:"telemetryEndpoints"`
	ProtocolID         string                 `json:"protocolId"`
	Genesis            Fields                 `json:"genesis"`
	Properties         map[string]interface{} `json:"properties"`
	ForkBlocks         []string               `json:"forkBlocks"`
	BadBlocks          []string               `json:"badBlocks"`
	ConsensusEngine    string                 `json:"consensusEngine"`
	CodeSubstitutes    map[string]string      `json:"codeSubstitutes"`
}

// Data defines the genesis file data formatted for trie storage
type Data struct {
	Name               string
	ID                 string
	ChainType          string
	Bootnodes          [][]byte
	TelemetryEndpoints []*TelemetryEndpoint
	ProtocolID         string
	Properties         map[string]interface{}
	ForkBlocks         []string
	BadBlocks          []string
	ConsensusEngine    string
	CodeSubstitutes    map[string]string
}

// TelemetryEndpoint struct to hold telemetry endpoint information
type TelemetryEndpoint struct {
	Endpoint  string `mapstructure:",squash"`
	Verbosity int    `mapstructure:",squash"`
}

// Fields stores genesis raw data, and human readable runtime data
type Fields struct {
	Raw     map[string]map[string]string      `json:"raw,omitempty"`
	Runtime map[string]map[string]interface{} `json:"runtime,omitempty"`
}

// GenesisData formats genesis for trie storage
func (g *Genesis) GenesisData() *Data {
	return &Data{
		Name:               g.Name,
		ID:                 g.ID,
		ChainType:          g.ChainType,
		Bootnodes:          common.StringArrayToBytes(g.Bootnodes),
		TelemetryEndpoints: interfaceToTelemetryEndpoint(g.TelemetryEndpoints),
		ProtocolID:         g.ProtocolID,
		Properties:         g.Properties,
		ForkBlocks:         g.ForkBlocks,
		BadBlocks:          g.BadBlocks,
		ConsensusEngine:    g.ConsensusEngine,
		CodeSubstitutes:    g.CodeSubstitutes,
	}
}

// GenesisFields returns the genesis fields including genesis raw data
func (g *Genesis) GenesisFields() Fields {
	return g.Genesis
}

// IsRaw returns whether the genesis is raw or not
func (g *Genesis) IsRaw() bool {
	return g.Genesis.Raw != nil || g.Genesis.Runtime == nil
}

// ToRaw converts a non-raw genesis to a raw genesis
func (g *Genesis) ToRaw() error {
	if g.IsRaw() {
		return nil
	}

	grt := g.Genesis.Runtime
	res, err := buildRawMap(grt)
	if err != nil {
		return err
	}

	g.Genesis.Raw = make(map[string]map[string]string)
	g.Genesis.Raw["top"] = res
	return nil
}

func interfaceToTelemetryEndpoint(endpoints []interface{}) []*TelemetryEndpoint {
	var res []*TelemetryEndpoint
	for _, v := range endpoints {
		epi, ok := v.([]interface{})
		if !ok {
			continue
		}
		if len(epi) != 2 {
			continue
		}
		eps, ok := epi[0].(string)
		if !ok {
			continue
		}
		epv, ok := epi[1].(float64)
		if !ok {
			continue
		}
		ep := &TelemetryEndpoint{
			Endpoint:  eps,
			Verbosity: int(epv),
		}
		res = append(res, ep)
	}

	return res
}
