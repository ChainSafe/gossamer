// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package candidatevalidation

func Register(overseerChan chan<- any) (*CandidateValidation, error) {
	candidateValidation := CandidateValidation{
		SubsystemToOverseer: overseerChan,
	}

	return &candidateValidation, nil
}
