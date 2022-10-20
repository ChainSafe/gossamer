// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package babe

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	// ErrBadSlotClaim is returned when a slot claim is invalid
	ErrBadSlotClaim = fmt.Errorf("could not verify slot claim VRF proof")

	// ErrBadSecondarySlotClaim is returned when a slot claim is invalid
	ErrBadSecondarySlotClaim = fmt.Errorf("invalid secondary slot claim")

	// ErrBadSignature is returned when a seal is invalid
	ErrBadSignature = fmt.Errorf("could not verify signature")

	// ErrProducerEquivocated is returned when a block producer has produced conflicting blocks
	ErrProducerEquivocated = fmt.Errorf("block producer equivocated")

	// ErrNotAuthorized is returned when the node is not authorized to produce a block
	ErrNotAuthorized = fmt.Errorf("not authorized to produce block")

	// ErrNoBABEHeader is returned when there is no BABE header found for a block, specifically when calculating randomness
	ErrNoBABEHeader = fmt.Errorf("no BABE header found for block")

	// ErrVRFOutputOverThreshold is returned when the vrf output for a block is invalid
	ErrVRFOutputOverThreshold = fmt.Errorf("vrf output over threshold")

	// ErrInvalidBlockProducerIndex is returned when the producer of a block isn't in the authority set
	ErrInvalidBlockProducerIndex = fmt.Errorf("block producer is not in authority set")

	// ErrAuthorityAlreadyDisabled is returned when attempting to disabled an already-disabled authority
	ErrAuthorityAlreadyDisabled = fmt.Errorf("authority has already been disabled")

	// ErrAuthorityDisabled is returned when attempting to verify a block produced by a disabled authority
	ErrAuthorityDisabled = fmt.Errorf("authority has been disabled for the remaining slots in the epoch")

	// ErrNotAuthority is returned when trying to perform authority functions when not an authority
	ErrNotAuthority = fmt.Errorf("node is not an authority")

	// ErrThresholdOneIsZero is returned when one of or both parameters to CalculateThreshold is zero
	ErrThresholdOneIsZero = fmt.Errorf("numerator or denominator cannot be 0")

	errNilParentHeader            = fmt.Errorf("parent header is nil")
	errInvalidResult              = fmt.Errorf("invalid error value")
	errFirstBlockTimeout          = fmt.Errorf("timed out waiting for first block")
	errChannelClosed              = fmt.Errorf("block notifier channel was closed")
	errOverPrimarySlotThreshold   = fmt.Errorf("cannot claim slot, over primary threshold")
	errNotOurTurnToPropose        = fmt.Errorf("cannot claim slot, not our turn to propose a block")
	errMissingDigestItems         = fmt.Errorf("block header is missing digest items")
	errServicePaused              = fmt.Errorf("service paused")
	errInvalidSlotTechnique       = fmt.Errorf("invalid slot claiming technique")
	errNoBABEAuthorityKeyProvided = fmt.Errorf("cannot create BABE service as authority; no keypair provided")
	errLastDigestItemNotSeal      = fmt.Errorf("last digest item is not seal")
	errLaggingSlot                = fmt.Errorf("current slot is smaller than slot of best block")

	other         Other
	invalidCustom InvalidCustom
	unknownCustom UnknownCustom
)

// A DispatchOutcomeError is outcome of dispatching the extrinsic
type DispatchOutcomeError struct {
	msg string // description of error
}

func (e DispatchOutcomeError) Error() string {
	return fmt.Sprintf("dispatch outcome error: %s", e.msg)
}

// A TransactionValidityError is possible errors while checking the validity of a transaction
type TransactionValidityError struct {
	msg error // description of error
}

func (e TransactionValidityError) Error() string {
	return fmt.Sprintf("transaction validity error: %s", e.msg)
}

var (
	errUnexpectedTxCall         = fmt.Errorf("call of the transaction is not expected")
	errInvalidPayment           = fmt.Errorf("invalid payment")
	errInvalidTransaction       = fmt.Errorf("invalid transaction")
	errOutdatedTransaction      = fmt.Errorf("outdated transaction")
	errBadProof                 = fmt.Errorf("bad proof")
	errAncientBirthBlock        = fmt.Errorf("ancient birth block")
	errExhaustsResources        = fmt.Errorf("exhausts resources")
	errMandatoryDispatchError   = fmt.Errorf("mandatory dispatch error")
	errInvalidMandatoryDispatch = fmt.Errorf("invalid mandatory dispatch")
	errLookupFailed             = fmt.Errorf("lookup failed")
	errValidatorNotFound        = fmt.Errorf("validator not found")
)

func newUnknownError(data scale.VaryingDataTypeValue) error {
	return fmt.Errorf("unknown error: %d", data)
}

// UnmarshalError occurs when unmarshalling fails
type UnmarshalError struct {
	msg string
}

func (e UnmarshalError) Error() string {
	return fmt.Sprintf("unmarshal error: %s", e.msg)
}

// Other Some error occurred
type Other string

// Index returns VDT index
func (Other) Index() uint { return 0 }

// CannotLookup Failed to lookup some data
type CannotLookup struct{}

// Index returns VDT index
func (CannotLookup) Index() uint { return 1 }

// BadOrigin A bad origin
type BadOrigin struct{}

// Index returns VDT index
func (BadOrigin) Index() uint { return 2 }

// Module A custom error in a module
type Module struct {
	Idx     uint8
	Err     uint8
	Message *string
}

// Index returns VDT index
func (Module) Index() uint { return 3 }

func (err Module) string() string {
	return fmt.Sprintf("index: %d code: %d message: %x", err.Idx, err.Err, *err.Message)
}

// ValidityCannotLookup Could not lookup some information that is required to validate the transaction
type ValidityCannotLookup struct{}

// Index returns VDT index
func (ValidityCannotLookup) Index() uint { return 0 }

// NoUnsignedValidator No validator found for the given unsigned transaction
type NoUnsignedValidator struct{}

// Index returns VDT index
func (NoUnsignedValidator) Index() uint { return 1 }

// UnknownCustom Any other custom unknown validity that is not covered
type UnknownCustom uint8

// Index returns VDT index
func (UnknownCustom) Index() uint { return 2 }

// Call The call of the transaction is not expected
type Call struct{}

// Index returns VDT index
func (Call) Index() uint { return 0 }

// Payment General error to do with the inability to pay some fees (e.g. account balance too low)
type Payment struct{}

// Index returns VDT index
func (Payment) Index() uint { return 1 }

// Future General error to do with the transaction not yet being valid (e.g. nonce too high)
type Future struct{}

// Index returns VDT index
func (Future) Index() uint { return 2 }

// Stale General error to do with the transaction being outdated (e.g. nonce too low)
type Stale struct{}

// Index returns VDT index
func (Stale) Index() uint { return 3 }

// BadProof General error to do with the transaction’s proofs (e.g. signature)
type BadProof struct{}

// Index returns VDT index
func (BadProof) Index() uint { return 4 }

// AncientBirthBlock The transaction birth block is ancient
type AncientBirthBlock struct{}

// Index returns VDT index
func (AncientBirthBlock) Index() uint { return 5 }

// ExhaustsResources The transaction would exhaust the resources of current block
type ExhaustsResources struct{}

// Index returns VDT index
func (ExhaustsResources) Index() uint { return 6 }

// InvalidCustom Any other custom invalid validity that is not covered
type InvalidCustom uint8

// Index returns VDT index
func (InvalidCustom) Index() uint { return 7 }

// BadMandatory An extrinsic with a Mandatory dispatch resulted in Error
type BadMandatory struct{}

// Index returns VDT index
func (BadMandatory) Index() uint { return 8 }

// MandatoryDispatch A transaction with a mandatory dispatch
type MandatoryDispatch struct{}

// Index returns VDT index
func (MandatoryDispatch) Index() uint { return 9 }

func determineErrType(vdt scale.VaryingDataType) error {
	vdtVal, err := vdt.Value()
	if err != nil {
		return fmt.Errorf("getting vdt value: %w", err)
	}
	switch val := vdtVal.(type) {
	case Other:
		return &DispatchOutcomeError{fmt.Sprintf("unknown error: %s", val)}
	case CannotLookup:
		return &DispatchOutcomeError{"failed lookup"}
	case BadOrigin:
		return &DispatchOutcomeError{"bad origin"}
	case Module:
		return &DispatchOutcomeError{fmt.Sprintf("custom module error: %s", val.string())}
	case Call:
		return &TransactionValidityError{errUnexpectedTxCall}
	case Payment:
		return &TransactionValidityError{errInvalidPayment}
	case Future:
		return &TransactionValidityError{errInvalidTransaction}
	case Stale:
		return &TransactionValidityError{errOutdatedTransaction}
	case BadProof:
		return &TransactionValidityError{errBadProof}
	case AncientBirthBlock:
		return &TransactionValidityError{errAncientBirthBlock}
	case ExhaustsResources:
		return &TransactionValidityError{errExhaustsResources}
	case InvalidCustom:
		return &TransactionValidityError{newUnknownError(val)}
	case BadMandatory:
		return &TransactionValidityError{errMandatoryDispatchError}
	case MandatoryDispatch:
		return &TransactionValidityError{errInvalidMandatoryDispatch}
	case ValidityCannotLookup:
		return &TransactionValidityError{errLookupFailed}
	case NoUnsignedValidator:
		return &TransactionValidityError{errValidatorNotFound}
	case UnknownCustom:
		return &TransactionValidityError{newUnknownError(val)}
	}

	return errInvalidResult
}

func determineErr(res []byte) error {
	dispatchError := scale.MustNewVaryingDataType(other, CannotLookup{}, BadOrigin{}, Module{})
	invalid := scale.MustNewVaryingDataType(Call{}, Payment{}, Future{}, Stale{}, BadProof{}, AncientBirthBlock{},
		ExhaustsResources{}, invalidCustom, BadMandatory{}, MandatoryDispatch{})
	unknown := scale.MustNewVaryingDataType(ValidityCannotLookup{}, NoUnsignedValidator{}, unknownCustom)

	okRes := scale.NewResult(nil, dispatchError)
	errRes := scale.NewResult(invalid, unknown)
	result := scale.NewResult(okRes, errRes)

	err := scale.Unmarshal(res, &result)
	if err != nil {
		return &UnmarshalError{err.Error()}
	}

	ok, err := result.Unwrap()
	if err != nil {
		switch o := err.(type) {
		case scale.WrappedErr:
			errResult := o.Err.(scale.Result)
			ok, err = errResult.Unwrap()
			if err != nil {
				switch err := err.(type) {
				case scale.WrappedErr:
					return determineErrType(err.Err.(scale.VaryingDataType))
				default:
					return errInvalidResult
				}
			} else {
				return determineErrType(ok.(scale.VaryingDataType))
			}
		default:
			return errInvalidResult
		}
	} else {
		switch o := ok.(type) {
		case scale.Result:
			_, err = o.Unwrap()
			if err != nil {
				switch err := err.(type) {
				case scale.WrappedErr:
					return determineErrType(err.Err.(scale.VaryingDataType))
				default:
					return errInvalidResult
				}
			} else {
				return nil
			}
		default:
			return errInvalidResult
		}
	}
}
