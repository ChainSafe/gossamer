// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

// This file is where the reputation change types are declared. Ones that require
// no calculation are defined as variables, while the ones that do (or take params)
// are defined as structs where the fields are the parameters needed to calculate
// the cost

// Costs/benefits that don't require calculation, example:

//var (
//	pastRejection = peerset.ReputationChange{
//		Value:  peerset.Reputation(-50),
//		Reason: "Grandpa: Past message",
//	}

//	invalidViewChange = peerset.ReputationChange{
//		Value:  peerset.Reputation(-500),
//		Reason: "Grandpa: Invalid view change",
//	}
//)

// Ones that do implement the cost function, example:

//type BadCommitMessage struct {
//	signaturesChecked   int
//	blocksLoaded        int
//	equivocationsCaught int
//}

//func (BadCommitMessage) cost() peerset.ReputationChange {
//	// TODO implement
//	return peerset.ReputationChange{}
//}
