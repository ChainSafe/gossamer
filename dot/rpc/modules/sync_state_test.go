package modules

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/genesis"
	"github.com/stretchr/testify/require"
)

const GssmrGenesisPath = "../../../chain/gssmr/genesis.json"

func TestSyncStateModule(t *testing.T) {
	module := NewSyncStateModule(GssmrGenesisPath)

	req := BoolRequest{
		Raw: true,
	}
	var res genesis.Genesis

	err := module.GenSyncSpec(nil, &req, &res)
	require.NoError(t, err)
}
