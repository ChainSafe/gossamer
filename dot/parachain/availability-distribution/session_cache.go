package availabilitydistribution

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
)

// SessionInfo is localised session information, tailored for the needs of availability distribution.
type SessionInfo struct {
	/// The index of this session.
	SessionIndex parachaintypes.SessionIndex

	/// Validator groups of the current session.
	///
	/// Each group's order is randomised. This way we achieve load balancing when requesting
	/// chunks, as the validators in a group will be tried in that randomised order. Each node
	/// should arrive at a different order, therefore we distribute the load on individual
	/// validators.
	ValidatorGroups [][]parachaintypes.AuthorityDiscoveryID

	/// Information about ourselves:
	OurIndex parachaintypes.ValidatorIndex

	/// Remember to which group we belong, so we won't start fetching chunks for candidates with
	/// our group being responsible. (We should have that chunk already.)
	///
	/// nil, if we are not in fact part of any group.
	OurGroup *parachaintypes.GroupIndex

	/// Node features.
	NodeFeatures parachaintypes.BitVec
}

func (s *SessionInfo) NumberOfValidators() uint {
	nValidators := uint(0)
	for _, group := range s.ValidatorGroups {
		nValidators += uint(len(group))
	}

	return nValidators
}

type SessionCache interface {
	GetSessionIndexForChild(parent common.Hash, rt runtime.Instance) (parachaintypes.SessionIndex, error)

	GetSessionInfo(
		sessionIndex parachaintypes.SessionIndex,
		rt runtime.Instance,
	) (*SessionInfo, error)

	GetAuthorityID(
		validatorIndex parachaintypes.ValidatorIndex,
		relayParent common.Hash,
		rt runtime.Instance,
	) (parachaintypes.AuthorityDiscoveryID, error)

	ReportBadValidators(
		sessionIndex parachaintypes.SessionIndex,
		groupIndex parachaintypes.GroupIndex,
		validators []parachaintypes.AuthorityDiscoveryID,
	) error
}

type LRUSessionCache struct {
	keystore keystore.Keystore

	// session index by relay parent
	sessionIndexCache *lrucache.LRUCache[common.Hash, *parachaintypes.SessionIndex]
	sessionInfoCache  *lrucache.LRUCache[parachaintypes.SessionIndex, *SessionInfo]
	// separate cache for GetAuthorityID(), maps relay parent hash to authority IDs indexable by validator index
	authIDCache *lrucache.LRUCache[common.Hash, []parachaintypes.AuthorityDiscoveryID]
}

const authIDCacheCapacity = 100

var _ SessionCache = (*LRUSessionCache)(nil)

func NewLRUSessionCache(keystore keystore.Keystore) *LRUSessionCache {
	return &LRUSessionCache{
		keystore: keystore,

		sessionIndexCache: lrucache.NewLRUCache[common.Hash, *parachaintypes.SessionIndex](10),
		// We need the current and previous session.
		sessionInfoCache: lrucache.NewLRUCache[parachaintypes.SessionIndex, *SessionInfo](2),
		authIDCache:      lrucache.NewLRUCache[common.Hash, []parachaintypes.AuthorityDiscoveryID](authIDCacheCapacity),
	}
}

// GetSessionIndexForChild returns the session index expected at any child of the `parent` block.
// This does not return the session index for the `parent` block.
func (c *LRUSessionCache) GetSessionIndexForChild(
	parent common.Hash,
	rt runtime.Instance,
) (parachaintypes.SessionIndex, error) {
	cachedIndex := c.sessionIndexCache.Get(parent)
	if cachedIndex != nil {
		return *cachedIndex, nil
	}

	sessionIndex, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return 0, err
	}

	c.sessionIndexCache.Put(parent, &sessionIndex)
	return sessionIndex, nil
}

// GetSessionInfo returns the session info for the given session index either from cache or from runtime information.
// If this node is not a validator, it returns nil.
func (c *LRUSessionCache) GetSessionInfo(
	sessionIndex parachaintypes.SessionIndex,
	rt runtime.Instance,
) (*SessionInfo, error) {
	if cachedSessionInfo := c.sessionInfoCache.Get(sessionIndex); cachedSessionInfo != nil {
		return cachedSessionInfo, nil
	}

	sessionInfo, err := rt.ParachainHostSessionInfo(sessionIndex)
	if err != nil {
		return nil, err
	}

	validatorID, ourIndex := parachainutil.SigningKeyAndIndex(sessionInfo.Validators, c.keystore)
	if validatorID == nil {
		// This node is not a validator.
		return nil, nil
	}

	nodeFeatures, err := rt.ParachainHostNodeFeatures()
	if err != nil {
		return nil, err
	}

	// Shuffle validator groups to load balance requests that will be sent to them later. If some of the validators
	// fail to respond correctly, they will be moved inside their group by ReportBadValidators() to ensure they are
	// tried last on subsequent requests.
	validatorGroups := c.convertAndShuffleValidatorGroups(
		sessionInfo.ValidatorGroups,
		sessionInfo.DiscoveryKeys,
		sessionIndex,
	)

	cachedSessionInfo := &SessionInfo{
		SessionIndex:    sessionIndex,
		ValidatorGroups: validatorGroups,
		OurIndex:        ourIndex,
		OurGroup:        c.getOurGroup(ourIndex, sessionInfo.ValidatorGroups),
		NodeFeatures:    nodeFeatures,
	}
	c.sessionInfoCache.Put(sessionIndex, cachedSessionInfo)
	return cachedSessionInfo, nil
}

func (c *LRUSessionCache) GetAuthorityID(
	validatorIndex parachaintypes.ValidatorIndex,
	relayParent common.Hash,
	rt runtime.Instance,
) (parachaintypes.AuthorityDiscoveryID, error) {
	authIDs := c.authIDCache.Get(relayParent)

	if authIDs == nil {
		sessionIndex, err := rt.ParachainHostSessionIndexForChild()
		if err != nil {
			return parachaintypes.AuthorityDiscoveryID{}, err
		}

		sessionInfo, err := rt.ParachainHostSessionInfo(sessionIndex)
		if err != nil {
			return parachaintypes.AuthorityDiscoveryID{}, err
		}

		authIDs = sessionInfo.DiscoveryKeys
		c.authIDCache.Put(relayParent, authIDs)
	}

	if int(validatorIndex) >= len(authIDs) {
		return parachaintypes.AuthorityDiscoveryID{}, fmt.Errorf("validator index %d is out of range", validatorIndex)
	}

	authID := authIDs[validatorIndex]
	return authID, nil
}

func (c *LRUSessionCache) getOurGroup(
	ourIndex parachaintypes.ValidatorIndex,
	groups [][]parachaintypes.ValidatorIndex,
) *parachaintypes.GroupIndex {
	for groupIndex, group := range groups {
		for _, validatorIndex := range group {
			if validatorIndex == ourIndex {
				g := parachaintypes.GroupIndex(groupIndex)
				return &g
			}
		}
	}
	return nil
}

// convertAndShuffleValidatorGroups converts validator indices to authority discovery IDs and shuffles each group.
func (c *LRUSessionCache) convertAndShuffleValidatorGroups(
	validatorGroups [][]parachaintypes.ValidatorIndex,
	discoveryKeys []parachaintypes.AuthorityDiscoveryID,
	sessionIndex parachaintypes.SessionIndex,
) [][]parachaintypes.AuthorityDiscoveryID {
	shuffledGroups := make([][]parachaintypes.AuthorityDiscoveryID, len(validatorGroups))

	for groupIndex, group := range validatorGroups {
		// Convert indices to authority IDs for this group.
		authIDs := make([]parachaintypes.AuthorityDiscoveryID, len(group))
		for j, validatorIndex := range group {
			authIDs[j] = discoveryKeys[int(validatorIndex)]
		}

		// Create a new random source using the session index and group index as seed.
		// This ensures different nodes get different shuffles.
		r := rand.New(rand.NewPCG(uint64(sessionIndex)*1000, uint64(groupIndex))) //nolint:gosec

		// Shuffle the group
		r.Shuffle(len(authIDs), func(i, j int) {
			authIDs[i], authIDs[j] = authIDs[j], authIDs[i]
		})

		shuffledGroups[groupIndex] = authIDs
	}
	return shuffledGroups
}

// ReportBadValidators ensures we try unresponsive or misbehaving validators last.
//
// We assume validators in a group are tried in reverse order, so the reported bad validators
// will be put at the beginning of the group.
func (c *LRUSessionCache) ReportBadValidators(
	sessionIndex parachaintypes.SessionIndex,
	groupIndex parachaintypes.GroupIndex,
	validators []parachaintypes.AuthorityDiscoveryID,
) error {
	session := c.sessionInfoCache.Get(sessionIndex)
	if session == nil {
		return errors.New("session not cached")
	}

	if int(groupIndex) >= len(session.ValidatorGroups) {
		return fmt.Errorf("invalid group index %d, number of groups: %d", groupIndex, len(session.ValidatorGroups))
	}

	group := session.ValidatorGroups[groupIndex]

	// Remove the bad ones from the group. This is inefficient but all slices we are working with are small.
	goodValidators := slices.DeleteFunc(group, func(v parachaintypes.AuthorityDiscoveryID) bool {
		return slices.Contains(validators, v)
	})

	// Create a new group by concatting the bad ones and the good ones. The bad ones go first because the group is
	// iterated in reverse order.
	session.ValidatorGroups[groupIndex] = append(validators, goodValidators...)
	return nil
}
