package runtime

// / An abstraction over justification for a block's validity under a consensus algorithm.
// /
// / Essentially a finality proof. The exact formulation will vary between consensus
// / algorithms. In the case where there are multiple valid proofs, inclusion within
// / the block itself would allow swapping justifications to change the block's hash
// / (and thus fork the chain). Sending a `Justification` alongside a block instead
// / bypasses this problem.
// /
// / Each justification is provided as an encoded blob, and is tagged with an ID
// / to identify the consensus engine that generated the proof (we might have
// / multiple justifications from different engines for the same block).
// pub type Justification = (ConsensusEngineId, EncodedJustification);
type Justification struct {
	ConsensusEngineID
	EncodedJustification
}

// / The encoded justification specific to a consensus engine.
// pub type EncodedJustification = Vec<u8>;
type EncodedJustification []byte

// / Collection of justifications for a given block, multiple justifications may
// / be provided by different consensus engines for the same block.
// pub struct Justifications(Vec<Justification>);
type Justifications []Justification

// IntoJustification returns a copy of the encoded justification for the given consensus
// engine, if it exists
func (j Justifications) IntoJustification(engineID ConsensusEngineID) *EncodedJustification {
	for _, justification := range j {
		if justification.ConsensusEngineID == engineID {
			return &justification.EncodedJustification
		}
	}
	return nil
}

// / Consensus engine unique ID.
// pub type ConsensusEngineId = [u8; 4];
type ConsensusEngineID [4]byte
