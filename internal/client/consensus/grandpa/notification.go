package grandpa

import "golang.org/x/exp/constraints"

// / The sending half of the Grandpa justification channel(s).
// /
// / Used to send notifications about justifications generated
// / at the end of a Grandpa round.
// pub type GrandpaJustificationSender<Block> = NotificationSender<GrandpaJustification<Block>>;
type GrandpaJustificationSender[
	Hash constraints.Ordered,
	N constraints.Unsigned,
	S comparable,
	ID AuthorityID,
] <-chan GrandpaJustification[
	Hash, N, S, ID,
	// TODO: shouldn't need to have to include this as generic param
	Header[Hash, N],
]
