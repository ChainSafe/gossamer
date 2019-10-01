package genesis

import (
	"reflect"
	"testing"
)

func TestParseJson(t *testing.T) {
	file := "../../genesis.json"
	genesis, err := ParseJson(file)
	if err != nil {
		t.Fatal(err)
	}

	expected := &Genesis{
		Name:       "gossamer",
		Id:         "gossamer",
		Bootnodes:  []string{"/ip4/104.211.54.233/tcp/30363/p2p/16Uiu2HAmFWPUx45xYYeCpAryQbvU3dY8PWGdMwS2tLm1dB1CsmCj"},
		ProtocolId: "gossamer",
		Genesis: genesisFields{
			Raw: []map[string]string{{"0x3a636f6465": "0x00"}},
		},
	}

	if !reflect.DeepEqual(expected, genesis) {
		t.Fatalf("Fail: expected %v got %v", expected, genesis)
	}
}
