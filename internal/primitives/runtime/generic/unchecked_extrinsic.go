package generic

// / Type to represent the version of the [Extension](TransactionExtension) used in this extrinsic.
// pub type ExtensionVersion = u8;
type ExtensionVersion uint8

// / Type to represent the extrinsic format version which defines an [UncheckedExtrinsic].
// pub type ExtrinsicVersion = u8;
type ExtrinsicVersion uint8

// / A "header" for extrinsics leading up to the call itself. Determines the type of extrinsic and
// / holds any necessary specialized data.
// #[derive(Eq, PartialEq, Clone)]
// pub enum Preamble<Address, Signature, Extension> {
type Preamble[Address, Signature, Extension any] interface {
	isPreamble()
}

// / An extrinsic without a signature or any extension. This means it's either an inherent or
// / an old-school "Unsigned" (we don't use that terminology any more since it's confusable with
// / the general transaction which is without a signature but does have an extension).
// /
// / NOTE: In the future, once we remove `ValidateUnsigned`, this will only serve Inherent
// / extrinsics and thus can be renamed to `Inherent`.
// Bare(ExtrinsicVersion),
type PreambleBare ExtrinsicVersion

func (pb PreambleBare) isPreamble() {}

// / An old-school transaction extrinsic which includes a signature of some hard-coded crypto.
// / Available only on extrinsic version 4.
// Signed(Address, Signature, Extension),
type PreambleSigned[Address, Signature, Extension any] struct {
	Address   Address
	Signature Signature
	Extension Extension
}

func (ps PreambleSigned[Address, Signature, Extension]) isPreamble() {}

// / A new-school transaction extrinsic which does not include a signature by default. The
// / origin authorization, through signatures or other means, is performed by the transaction
// / extension in this extrinsic. Available starting with extrinsic version 5.
// General(ExtensionVersion, Extension),
type PreambleGeneral[Extension any] struct {
	ExtensionVersion ExtensionVersion
	Extension        Extension
}

func (pg PreambleGeneral[Extension]) isPreamble() {}

// pub struct UncheckedExtrinsic<Address, Call, Signature, Extension> {
type UncheckedExtrinsic[Address, Call, Signature, Extension any] struct {
	/// Information regarding the type of extrinsic this is (inherent or transaction) as well as
	/// associated extension (`Extension`) data if it's a transaction and a possible signature.
	// pub preamble: Preamble<Address, Signature, Extension>,
	Preamble Preamble[Address, Signature, Extension]
	/// The function that should be called.
	// pub function: Call,
	Function Call
}

func (ue UncheckedExtrinsic[Address, Call, Signature, Extension]) IsSigned() *bool {
	switch ue.Preamble.(type) {
	case PreambleSigned[Address, Signature, Extension]:
		t := true
		return &t
	case PreambleGeneral[Extension]:
		f := false
		return &f
	case PreambleBare:
		f := false
		return &f
	default:
		return nil
	}
}

func (UncheckedExtrinsic[Address, Call, Signature, Extension]) Bytes() []byte {
	panic("unimpl")
}
