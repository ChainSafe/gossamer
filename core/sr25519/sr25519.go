package sr25519

// / An Schnorrkel/Ristretto x25519 ("sr25519") public key.
type Public [32]byte

// / An Schnorrkel/Ristretto x25519 ("sr25519") signature.
type Signature [64]byte

// / Transcript ready to be used for VRF related operations.
// #[derive(Clone)]
// pub struct VrfTranscript(pub merlin::Transcript);
type VRFTranscript struct {
}

// / VRF input ready to be used for VRF sign and verify operations.
type VRFSignData struct {
	/// Transcript data contributing to VRF output.
	transcript VRFTranscript
	/// Extra transcript data to be signed by the VRF.
	extra *VRFTranscript
}

// /// Transcript ready to be used for VRF related operations.
// #[derive(Clone)]
// pub struct VrfTranscript(pub merlin::Transcript);

// impl VrfTranscript {
// 	/// Build a new transcript instance.
// 	///
// 	/// Each `data` element is a tuple `(domain, message)` composing the transcipt.
// 	pub fn new(label: &'static [u8], data: &[(&'static [u8], &[u8])]) -> Self {
// 		let mut transcript = merlin::Transcript::new(label);
// 		data.iter().for_each(|(l, b)| transcript.append_message(l, b));
// 		VrfTranscript(transcript)
// 	}

// 	/// Map transcript to `VrfSignData`.
// 	pub fn into_sign_data(self) -> VrfSignData {
// 		self.into()
// 	}
// }

// VRFInput is a transcript used by the Fiat-Shamir transform.
type VRFInput VRFTranscript

// / VRF output type suitable for schnorrkel operations.
// #[derive(Clone, Debug, PartialEq, Eq)]
// pub struct VrfOutput(pub schnorrkel::vrf::VRFOutput);
type VRFOutput struct{}

// /// VRF proof type suitable for schnorrkel operations.
// #[derive(Clone, Debug, PartialEq, Eq)]
// pub struct VrfProof(pub schnorrkel::vrf::VRFProof);
type VRFProof struct{}

// / VRF signature data
type VRFSignature struct {
	Output VRFOutput
	Proof  VRFProof
}
