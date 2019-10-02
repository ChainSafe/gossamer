package genesis

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

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
