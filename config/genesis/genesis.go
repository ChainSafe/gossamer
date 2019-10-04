package genesis

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/trie"
)

// Genesis stores the data parsed from the genesis configuration file
type Genesis struct {
	Name       string
	Id         string
	Bootnodes  []string
	ProtocolId string
	Genesis    genesisFields
}

type genesisFields struct {
	Raw []map[string]string
}

// ParseJson parses a JSON formatted genesis file
func ParseJson(file string) (*Genesis, error) {
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

// GenesisState stores the genesis state after it's been loaded into a trie and network configuartion
type GenesisState struct {
	Name        string
	Id          string
	GenesisTrie *trie.Trie
	P2pConfig   *p2p.Config
}
