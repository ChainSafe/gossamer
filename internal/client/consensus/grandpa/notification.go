// Copyright 2025 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/internal/client/utils/notification"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
)

// The sending half of the Grandpa justification channel(s).
//
// Used to send notifications about justifications generated at the end of a Grandpa round.
type GrandpaJustificationSender[Hash runtime.Hash, N runtime.Number, Header runtime.Header[N, Hash]] struct {
	notification.NotificationSender[GrandpaJustification[Hash, N, Header]]
}

// / The receiving half of the Grandpa justification channel.
// /
// / Used to receive notifications about justifications generated
// / at the end of a Grandpa round.
// / The `GrandpaJustificationStream` entity stores the `SharedJustificationSenders`
// / so it can be used to add more subscriptions.
// pub type GrandpaJustificationStream<Block> =
//
//	NotificationStream<GrandpaJustification<Block>, GrandpaJustificationsTracingKey>;
type GrandpaJustificationStream[Hash runtime.Hash, N runtime.Number, Header runtime.Header[N, Hash]] struct {
	notification.NotificationStream[GrandpaJustification[Hash, N, Header]]
}
