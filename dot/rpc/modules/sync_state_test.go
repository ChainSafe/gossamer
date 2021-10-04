package modules

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const GssmrGenesisPath = "../../../chain/gssmr/genesis.json"

func TestSyncStateModule(t *testing.T) {
	module := NewSyncStateModule(SyncState{GenesisFilePath: GssmrGenesisPath})

	req := true
	var res []byte

	err := module.GenSyncSpec(nil, &req, &res)
	require.NoError(t, err)

	fmt.Println(string(res))
}
