package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/genesis"

	"github.com/stretchr/testify/require"
)

func TestCreateGenesisWithRuntime(t *testing.T) {
	defaultGenesisSpecPath = "../../chain/gssmr/genesis-spec.json"

	testCode := []byte("somecode")
	testHex := common.BytesToHex(testCode)
	testFile, err := ioutil.TempFile("", "testcode-*.wasm")
	require.NoError(t, err)
	defer os.Remove(testFile.Name())

	err = ioutil.WriteFile(testFile.Name(), testCode, 0777)
	require.NoError(t, err)

	out, err := createGenesisWithRuntime(testFile.Name())
	require.NoError(t, err)

	g := new(genesis.Genesis)
	err = json.Unmarshal([]byte(out), g)
	require.NoError(t, err)
	require.Equal(t, testHex, g.Genesis.Runtime["System"]["code"].(string))
}
