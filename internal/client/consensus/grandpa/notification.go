package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"golang.org/x/exp/constraints"
)

// / The sending half of the Grandpa justification channel(s).
// /
// / Used to send notifications about justifications generated
// / at the end of a Grandpa round.
// pub type GrandpaJustificationSender<Block> = NotificationSender<GrandpaJustification<Block>>;
type GrandpaJustificationSender[Hash constraints.Ordered, N runtime.Number] <-chan GrandpaJustification[Hash, N]
