package availabilitydistribution

import (
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// Localised session information, tailored for the needs of availability distribution.
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
	GetSessionInfo(
		sessionIndex parachaintypes.SessionIndex,
		rt runtime.Instance,
	) (*SessionInfo, error)

	ReportBadValidators(
		sessionIndex parachaintypes.SessionIndex,
		groupIndex parachaintypes.GroupIndex,
		validators []parachaintypes.AuthorityDiscoveryID,
	) error
}

// TODO implement the interface (#4494)
