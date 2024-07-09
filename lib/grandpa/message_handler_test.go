package grandpa

import (
	"testing"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/ed25519"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVerify_WestendBlock512_Justification(t *testing.T) {
	wndSetID0Voters := make([]types.GrandpaVoter, 0)
	wndSetID0Authorities := []string{
		"0x959cebf18fecb305b96fd998c95f850145f52cbbb64b3ef937c0575cc7ebd652",
		"0x9fc415cce1d0b2eed702c9e05f476217d23b46a8723fd56f08cddad650be7c2d",
		"0xfeca0be2c87141f6074b221c919c0161a1c468d9173c5c1be59b68fab9a0ff93",
	}

	for idx, pubkey := range wndSetID0Authorities {
		edPubKey, err := ed25519.NewPublicKey(common.MustHexToBytes(pubkey))
		require.NoError(t, err)

		wndSetID0Voters = append(wndSetID0Voters, types.GrandpaVoter{
			ID:  uint64(idx),
			Key: *edPubKey,
		})
	}

	const currentSetID uint64 = 0
	const block512Justification = "0xc9020000000000005895897f12e1a670609929433ac7a69dcae90e0cc2d9c" +
		"32c0dce0e2a5e5e614e000200000c5895897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e" +
		"614e000200006216ec969bb5133b13f54a6121ef3a908d0a87d8409e2d471c0cad1c28532b6e27d6a8d746b43" +
		"df96c2149915252a846227b060372e3bb6f49e91500d3d8ef0d959cebf18fecb305b96fd998c95f850145f52c" +
		"bbb64b3ef937c0575cc7ebd6525895897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e614" +
		"e0002000092820b93ac482089fffc8246b4111da2e2b7adc786938c24eb25fe3b97cd21b946b7e12cb6fa5546" +
		"b73c047ffc7c73b17a6a750bc6f2858bb0d0a7fff2fdd2029fc415cce1d0b2eed702c9e05f476217d23b46a87" +
		"23fd56f08cddad650be7c2d5895897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e614e00" +
		"02000017a338b777152d2213908ab29f961ebbca04e6bd1e4cfde6cb1a0b7b7f244c2670935cdf4c2acb4dd06" +
		"1913848f5865aa887406a3ea0c8d0dcd4d551ff249900feca0be2c87141f6074b221c919c0161a1c468d9173c5c1be59b68fab9a0ff9300"

	ctrl := gomock.NewController(t)
	grandpaMockService := NewMockGrandpaState(ctrl)
	grandpaMockService.EXPECT().GetSetIDByBlockNumber(uint(512)).Return(currentSetID, nil)
	grandpaMockService.EXPECT().GetAuthorities(currentSetID).Return(wndSetID0Voters, nil)

	service := &Service{
		grandpaState: grandpaMockService,
	}

	round, setID, err := service.VerifyBlockJustification(
		common.MustHexToHash("0x5895897f12e1a670609929433ac7a69dcae90e0cc2d9c32c0dce0e2a5e5e614e"),
		512,
		common.MustHexToBytes(block512Justification))

	require.NoError(t, err)
	require.Equal(t, uint64(0), setID)
	require.Equal(t, uint64(713), round)
}
