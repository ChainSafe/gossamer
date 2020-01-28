package genesis

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/ChainSafe/gossamer/common"
)

// Genesis stores the data parsed from the genesis configuration file
type Genesis struct {
	Name       string
	ID         string
	Bootnodes  []string
	ProtocolID string
	Genesis    GenesisFields
}

// GenesisData stores the data parsed from the genesis configuration file
type GenesisData struct {
	Name          string
	ID            string
	Bootnodes     [][]byte
	ProtocolID    string
	genesisFields GenesisFields
}

// GenesisFields struct
type GenesisFields struct {
	Raw map[string]string
}

// LoadGenesisJSONFile parses a JSON formatted genesis file
func LoadGenesisJSONFile(file string) (*Genesis, error) {
	fp, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	g := new(Genesis)
	err = json.Unmarshal(data, g)
	return g, err
}

// LoadGenesisData returns GenesisData
func LoadGenesisData(file string) (*GenesisData, error) {
	g, err := LoadGenesisJSONFile(file)
	if err != nil {
		return nil, err
	}

	return &GenesisData{
		Name:          g.Name,
		ID:            g.ID,
		Bootnodes:     common.StringArrayToBytes(g.Bootnodes),
		ProtocolID:    g.ProtocolID,
		genesisFields: g.Genesis,
	}, nil
}

// GenesisFields returns genesisFields struct
func (g *GenesisData) GenesisFields() GenesisFields {
	return g.genesisFields
}
