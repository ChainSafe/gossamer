package availabilitydistribution

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReportBadValidators(t *testing.T) {
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
			sessionCache := NewLRUSessionCache(nil)

			for sessionIndex, sessionInfo := range tc.cacheContent {
				sessionCache.sessionInfoCache.Put(sessionIndex, sessionInfo)

				err := sessionCache.ReportBadValidators(sessionIndex, groupIndex, badValidators)
				if tc.errExpected {
					require.Error(t, err)
					continue
				}
				require.NoError(t, err)

				sessionInfo, err := sessionCache.GetSessionInfo(sessionIndex, nil)
				require.NoError(t, err)

				group := sessionInfo.ValidatorGroups[groupIndex]
				bad, good := group[:len(badValidators)], group[len(badValidators):]
				require.Equal(t, bad, badValidators)
				require.Equal(t, good, validatorGroups[groupIndex][len(badValidators):])
			}
		})
	}
}
