package genesis

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestParseGenesisJson(t *testing.T) {
	// Create temp file
	file, err := ioutil.TempFile("", "genesis-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	expected := &Genesis{
		Name:       "gossamer",
		Id:         "gossamer",
		Bootnodes:  []string{"/ip4/104.211.54.233/tcp/30363/p2p/16Uiu2HAmFWPUx45xYYeCpAryQbvU3dY8PWGdMwS2tLm1dB1CsmCj"},
		ProtocolId: "gossamer",
		Genesis:    GenesisFields{},
	}

	testBytes, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	testHex := hex.EncodeToString(testBytes)
	testRaw := [2]map[string]string{}
	testRaw[0] = map[string]string{"0x3a636f6465": "0x" + testHex}
	expected.Genesis = GenesisFields{Raw: testRaw}

	// Grab json encoded bytes
	bz, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	// Write to temp file
	_, err = file.Write(bz)
	if err != nil {
		t.Fatal(err)
	}

	genesis, err := LoadGenesisJSONFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expected, genesis) {
		t.Fatalf("Fail: expected %v got %v", expected, genesis)
	}
}
