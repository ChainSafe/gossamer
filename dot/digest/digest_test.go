package digest

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/state"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T) (*Handler, *state.Service) {
	testDatadirPath := t.TempDir()

	ctrl := gomock.NewController(t)
	telemetryMock := NewMockTelemetry(ctrl)
	telemetryMock.EXPECT().SendMessage(gomock.Any()).AnyTimes()

	config := state.Config{
		Path:      testDatadirPath,
		Telemetry: telemetryMock,
	}
	stateSrvc := state.NewService(config)
	stateSrvc.UseMemDB()

	gen, genesisTrie, genesisHeader := newWestendDevGenesisWithTrieAndHeader(t)
	err := stateSrvc.Initialise(&gen, &genesisHeader, &genesisTrie)
	require.NoError(t, err)

	err = stateSrvc.SetupBase()
	require.NoError(t, err)

	err = stateSrvc.Start()
	require.NoError(t, err)

	dh, err := NewHandler(log.Critical, stateSrvc.Block, stateSrvc.Epoch, stateSrvc.Grandpa)
	require.NoError(t, err)
	return dh, stateSrvc
}

func TestDigestHashes(t *testing.T) {

	babePreRuntimeDigest := types.NewBABEPreRuntimeDigest(common.MustHexToBytes("0x02020000002fe4d90f00000000"))
	grandpaConsensus1 := types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              common.MustHexToBytes("0x0118a8ddd0891e14725841cd1b5581d23806a97f41c28a25436db6473c86e15dcd4f01000000000000007ca58770eb41c1a68ef77e92255e4635fc11f665cb89aee469e920511c48343a010000000000000074bfb70627416e6e6c4785e928ced384c6c06e5c8dd173a094bc3118da7b673e0100000000000000d455d6778e7100787f0e51e42b86e6e3aac96b1f68aaab59678ab1dd28e5374f0100000000000000a694eb96e1674003ccff3309937bc3ab62ad1a66436f5b1dfad03fc81e8a4f700100000000000000786fc9c50f5d26a2c9f8028fc31f1a447d3425349eb5733550201c68e495a22d01000000000000005eee23b75c97a69e537632302d88870a0f48c05d6a3b11aeb5d3fdf8b579ba79"),
	}
	grandpaConsensus2 := types.ConsensusDigest{
		ConsensusEngineID: types.GrandpaEngineID,
		Data:              common.MustHexToBytes("0x02c59e1500189fc415cce1d0b2eed702c9e05f476217d23b46a8723fd56f08cddad650be7c2d0100000000000000feca0be2c87141f6074b221c919c0161a1c468d9173c5c1be59b68fab9a0ff930100000000000000fc9d33059580a69454179ffa41cbae6de2bc8d2bd2c3f1d018fe5484a5a919560100000000000000059ddb0eb77615669a1fc7962bbff119c20c18b58b4922788f842f3cd5b2813a010000000000000007d952daf2d0e2616e5344a6cff989a3fcc5a79a5799198c15ff1c06c51a1280010000000000000065c30e319f817c4392a7c2b98f1585541d53bf8d096bd64033cce6bacbde2952010000000000000005000000"),
	}

	digests := types.NewDigest()
	err := digests.Add(babePreRuntimeDigest)
	require.NoError(t, err)
	err = digests.Add(grandpaConsensus1)
	require.NoError(t, err)
	err = digests.Add(grandpaConsensus2)
	require.NoError(t, err)

	header := types.NewHeader(common.Hash{}, common.Hash{}, common.Hash{}, 1, digests)

	handler, _ := newTestHandler(t)
	err = handler.HandleDigests(header)
	require.NoError(t, err)

	err = handler.grandpaState.ApplyForcedChanges(header)
	require.NoError(t, err)
}
