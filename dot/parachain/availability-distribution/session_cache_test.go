package availabilitydistribution

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestReportBadValidators(t *testing.T) {
	t.Parallel()

	// predefine some test data to make the test cases more readable
	sessionIndex := parachaintypes.SessionIndex(1)
	groupIndex := parachaintypes.GroupIndex(0)

	validatorGroups := [][]parachaintypes.AuthorityDiscoveryID{
		{
			parachaintypes.AuthorityDiscoveryID{0x03},
			parachaintypes.AuthorityDiscoveryID{0x07},
			parachaintypes.AuthorityDiscoveryID{0x04},
			parachaintypes.AuthorityDiscoveryID{0x06},
		},
		{
			parachaintypes.AuthorityDiscoveryID{0x02},
			parachaintypes.AuthorityDiscoveryID{0x01},
			parachaintypes.AuthorityDiscoveryID{0x08},
			parachaintypes.AuthorityDiscoveryID{0x05},
		},
	}

	sessionInfo := &SessionInfo{
		ValidatorGroups: validatorGroups,
	}

	badValidators := []parachaintypes.AuthorityDiscoveryID{{0x04}, {0x07}}

	testCases := []struct {
		description   string
		cacheContent  map[parachaintypes.SessionIndex]*SessionInfo
		sessionIndex  parachaintypes.SessionIndex
		groupIndex    parachaintypes.GroupIndex
		badValidators []parachaintypes.AuthorityDiscoveryID
		errExpected   bool
	}{
		{
			description:   "session_not_cached",
			cacheContent:  map[parachaintypes.SessionIndex]*SessionInfo{},
			sessionIndex:  sessionIndex,
			groupIndex:    groupIndex,
			badValidators: badValidators,
			errExpected:   true,
		},
		{
			description: "invalid_group_index",
			cacheContent: map[parachaintypes.SessionIndex]*SessionInfo{
				sessionIndex: sessionInfo,
			},
			sessionIndex:  sessionIndex,
			groupIndex:    parachaintypes.GroupIndex(5),
			badValidators: badValidators,
			errExpected:   true,
		},
		{
			description: "happy_path",
			cacheContent: map[parachaintypes.SessionIndex]*SessionInfo{
				sessionIndex: sessionInfo,
			},
			sessionIndex:  sessionIndex,
			groupIndex:    groupIndex,
			badValidators: badValidators,
			errExpected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			sessionCache := NewLRUSessionCache(nil)

			for sessionIndex, sessionInfo := range tc.cacheContent {
				sessionCache.sessionInfoCache.Put(sessionIndex, sessionInfo)

				err := sessionCache.ReportBadValidators(sessionIndex, tc.groupIndex, tc.badValidators)
				if tc.errExpected {
					require.Error(t, err)
					continue
				}
				require.NoError(t, err)

				sessionInfo, err := sessionCache.GetSessionInfo(sessionIndex, nil)
				require.NoError(t, err)

				group := sessionInfo.ValidatorGroups[groupIndex]
				bad, good := group[:len(tc.badValidators)], group[len(tc.badValidators):]

				expectedGood := []parachaintypes.AuthorityDiscoveryID{{0x03}, {0x06}}

				require.Equal(t, tc.badValidators, bad)
				require.Equal(t, expectedGood, good)
			}
		})
	}
}

func TestConvertAndShuffleValidatorGroups(t *testing.T) {
	t.Parallel()

	validatorGroups := [][]parachaintypes.ValidatorIndex{
		{5, 2, 8}, // group index 0
		{1, 7, 4}, // group index 1
		{3, 0, 6}, // group index 2
	}

	discoveryKeys := []parachaintypes.AuthorityDiscoveryID{
		{0x00}, // validator index 0, group index 2
		{0x01}, // validator index 1, group index 1
		{0x02}, // validator index 2, group index 0
		{0x03}, // validator index 3, group index 2
		{0x04}, // validator index 4, group index 1
		{0x05}, // validator index 5, group index 0
		{0x06}, // validator index 6, group index 2
		{0x07}, // validator index 7, group index 1
		{0x08}, // validator index 8, group index 0
	}

	cache := NewLRUSessionCache(nil)

	converted := cache.convertAndShuffleValidatorGroups(
		validatorGroups,
		discoveryKeys,
		parachaintypes.SessionIndex(1),
	)

	// group index 0, discovery keys of validator indices 5, 2, 8 need to be present
	require.Contains(t, converted[0], parachaintypes.AuthorityDiscoveryID{0x05})
	require.Contains(t, converted[0], parachaintypes.AuthorityDiscoveryID{0x02})
	require.Contains(t, converted[0], parachaintypes.AuthorityDiscoveryID{0x08})

	// group index 1, discovery keys of validator indices 1, 7, 4 need to be present
	require.Contains(t, converted[1], parachaintypes.AuthorityDiscoveryID{0x01})
	require.Contains(t, converted[1], parachaintypes.AuthorityDiscoveryID{0x07})
	require.Contains(t, converted[1], parachaintypes.AuthorityDiscoveryID{0x04})

	// group index 2, discovery keys of validator indices 3, 0, 6 need to be present
	require.Contains(t, converted[2], parachaintypes.AuthorityDiscoveryID{0x03})
	require.Contains(t, converted[2], parachaintypes.AuthorityDiscoveryID{0x00})
	require.Contains(t, converted[2], parachaintypes.AuthorityDiscoveryID{0x06})
}

func TestGetOurGroup(t *testing.T) {
	t.Parallel()

	expectedGroupIndex := parachaintypes.GroupIndex(1)

	testCases := []struct {
		description string
		ourIndex    parachaintypes.ValidatorIndex
		groups      [][]parachaintypes.ValidatorIndex
		expected    *parachaintypes.GroupIndex
	}{
		{
			description: "has_group",
			ourIndex:    parachaintypes.ValidatorIndex(4),
			groups: [][]parachaintypes.ValidatorIndex{
				{parachaintypes.ValidatorIndex(0), parachaintypes.ValidatorIndex(1), parachaintypes.ValidatorIndex(2)},
				{parachaintypes.ValidatorIndex(3), parachaintypes.ValidatorIndex(4), parachaintypes.ValidatorIndex(5)},
				{parachaintypes.ValidatorIndex(6), parachaintypes.ValidatorIndex(7), parachaintypes.ValidatorIndex(8)},
			},
			expected: &expectedGroupIndex,
		},
		{
			description: "has_no_group",
			ourIndex:    parachaintypes.ValidatorIndex(20),
			groups: [][]parachaintypes.ValidatorIndex{
				{parachaintypes.ValidatorIndex(0), parachaintypes.ValidatorIndex(1), parachaintypes.ValidatorIndex(2)},
				{parachaintypes.ValidatorIndex(3), parachaintypes.ValidatorIndex(4), parachaintypes.ValidatorIndex(5)},
				{parachaintypes.ValidatorIndex(6), parachaintypes.ValidatorIndex(7), parachaintypes.ValidatorIndex(8)},
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			cache := NewLRUSessionCache(nil)

			group := cache.getOurGroup(tc.ourIndex, tc.groups)

			require.Equal(t, tc.expected, group)
		})
	}
}

func TestGetSessionInfo(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	keypairs := generateKeypairs(t, 12)
	validators := validatorIDsFromKeypairs(t, keypairs)
	ourID, ourIndex := validators[7], parachaintypes.ValidatorIndex(7)

	keystoreMock := setUpKeystoreMock(t, ctrl, ourID, keypairs[ourIndex])

	validatorGroups := [][]parachaintypes.ValidatorIndex{
		{5, 2, 8},   // group index 0
		{1, 7, 4},   // group index 1 -> our group
		{3, 0, 6},   // group index 2
		{10, 11, 9}, // group index 3
	}

	discoveryKeys := generateDiscoveryKeys(t, len(validators))
	sessionIndex := parachaintypes.SessionIndex(3)

	nodeFeatures, err := parachaintypes.NewBitVec([]bool{false, false, false, false})
	require.NoError(t, err)

	runtimeMock := setUpRuntimeMock(t, ctrl, sessionIndex, validators, discoveryKeys, validatorGroups, nodeFeatures)

	cache := NewLRUSessionCache(keystoreMock)

	sessionInfo, err := cache.GetSessionInfo(sessionIndex, runtimeMock)
	require.NoError(t, err)
	require.NotNil(t, sessionInfo)

	require.Equal(t, sessionIndex, sessionInfo.SessionIndex)
	require.Equal(t, ourIndex, sessionInfo.OurIndex)

	ourExpectedGroupIndex := parachaintypes.GroupIndex(1)
	require.Equal(t, &ourExpectedGroupIndex, sessionInfo.OurGroup)

	require.Equal(t, nodeFeatures, sessionInfo.NodeFeatures)
	require.Len(t, sessionInfo.ValidatorGroups, len(validatorGroups))

	ourDiscoveryKey := discoveryKeys[ourIndex]
	ourGroup := sessionInfo.ValidatorGroups[*sessionInfo.OurGroup]
	require.Contains(t, ourGroup, ourDiscoveryKey)
}

func TestGetSessionIndexForChild(t *testing.T) {
	t.Parallel()

	parent := common.Hash{0x01}
	sessionIndex := parachaintypes.SessionIndex(5)

	t.Run("from_cache", func(t *testing.T) {
		t.Parallel()

		cache := NewLRUSessionCache(nil)
		cache.sessionIndexCache.Put(parent, &sessionIndex)

		index, err := cache.GetSessionIndexForChild(parent, nil)

		require.NoError(t, err)
		require.Equal(t, sessionIndex, index)
	})

	t.Run("from_runtime", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		rt := NewMockInstance(ctrl)
		rt.EXPECT().
			ParachainHostSessionIndexForChild().
			Return(sessionIndex, nil).
			Times(1)

		cache := NewLRUSessionCache(nil)

		index, err := cache.GetSessionIndexForChild(parent, rt)

		require.NoError(t, err)
		require.Equal(t, sessionIndex, index)

		// Check that the cache was updated
		index, err = cache.GetSessionIndexForChild(parent, nil)

		require.NoError(t, err)
		require.Equal(t, sessionIndex, index)
	})
}

func generateKeypairs(t *testing.T, n int) []*sr25519.Keypair {
	t.Helper()

	kps := make([]*sr25519.Keypair, n)
	for i := 0; i < n; i++ {
		kp, err := sr25519.GenerateKeypair()
		require.NoError(t, err)
		kps[i] = kp
	}
	return kps
}

func validatorIDsFromKeypairs(t *testing.T, kps []*sr25519.Keypair) []parachaintypes.ValidatorID {
	t.Helper()

	validatorIDs := make([]parachaintypes.ValidatorID, len(kps))
	for i, kp := range kps {
		validatorIDs[i] = parachaintypes.ValidatorID(kp.Public().Encode())
	}
	return validatorIDs
}

func generateDiscoveryKeys(t *testing.T, n int) []parachaintypes.AuthorityDiscoveryID {
	t.Helper()

	discoveryKeys := make([]parachaintypes.AuthorityDiscoveryID, n)
	for i := 0; i < n; i++ {
		var k parachaintypes.AuthorityDiscoveryID
		k[0] = byte(i)
		k[1] = byte(i * 2)
		discoveryKeys[i] = k
	}
	return discoveryKeys
}

func setUpKeystoreMock(
	t *testing.T,
	ctrl *gomock.Controller,
	validatorID parachaintypes.ValidatorID,
	keypair *sr25519.Keypair,
) keystore.Keystore {
	t.Helper()

	ks := NewMockKeystore(ctrl)

	ourPubkey, err := sr25519.NewPublicKey(validatorID[:])
	require.NoError(t, err)

	ks.EXPECT().
		GetKeypair(gomock.Any()).
		DoAndReturn(func(pubkey *sr25519.PublicKey) keystore.KeyPair {
			if pubkey.Hex() == ourPubkey.Hex() {
				return keypair
			}
			return nil
		}).
		AnyTimes()

	return ks
}

func setUpRuntimeMock(
	t *testing.T,
	ctrl *gomock.Controller,
	sessionIndex parachaintypes.SessionIndex,
	validators []parachaintypes.ValidatorID,
	discoveryKeys []parachaintypes.AuthorityDiscoveryID,
	validatorGroups [][]parachaintypes.ValidatorIndex,
	nodeFeatures parachaintypes.BitVec,
) runtime.Instance {
	t.Helper()

	runtimeMock := NewMockInstance(ctrl)

	runtimeMock.EXPECT().
		ParachainHostSessionInfo(gomock.AssignableToTypeOf(sessionIndex)).
		Return(&parachaintypes.SessionInfo{
			Validators:      validators,
			DiscoveryKeys:   discoveryKeys,
			ValidatorGroups: validatorGroups,
		}, nil).
		Times(1)

	runtimeMock.EXPECT().
		ParachainHostNodeFeatures().
		Return(nodeFeatures, nil).
		Times(1)

	return runtimeMock
}
