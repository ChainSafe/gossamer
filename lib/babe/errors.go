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
	"errors"
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	// ErrBadSlotClaim is returned when a slot claim is invalid
	ErrBadSlotClaim = errors.New("could not verify slot claim VRF proof")

	// ErrBadSecondarySlotClaim is returned when a slot claim is invalid
	ErrBadSecondarySlotClaim = errors.New("invalid secondary slot claim")

	// ErrBadSignature is returned when a seal is invalid
	ErrBadSignature = errors.New("could not verify signature")

	// ErrProducerEquivocated is returned when a block producer has produced conflicting blocks
	ErrProducerEquivocated = errors.New("block producer equivocated")

	// ErrNotAuthorized is returned when the node is not authorized to produce a block
	ErrNotAuthorized = errors.New("not authorized to produce block")

	// ErrNoBABEHeader is returned when there is no BABE header found for a block, specifically when calculating randomness
	ErrNoBABEHeader = errors.New("no BABE header found for block")

	// ErrVRFOutputOverThreshold is returned when the vrf output for a block is invalid
	ErrVRFOutputOverThreshold = errors.New("vrf output over threshold")

	// ErrInvalidBlockProducerIndex is returned when the producer of a block isn't in the authority set
	ErrInvalidBlockProducerIndex = errors.New("block producer is not in authority set")

	// ErrAuthorityAlreadyDisabled is returned when attempting to disabled an already-disabled authority
	ErrAuthorityAlreadyDisabled = errors.New("authority has already been disabled")

	// ErrAuthorityDisabled is returned when attempting to verify a block produced by a disabled authority
	ErrAuthorityDisabled = errors.New("authority has been disabled for the remaining slots in the epoch")

	// ErrNotAuthority is returned when trying to perform authority functions when not an authority
	ErrNotAuthority = errors.New("node is not an authority")

	errNilBlockImportHandler = errors.New("cannot have nil BlockImportHandler")
	errNilBlockState         = errors.New("cannot have nil BlockState")
	errNilEpochState         = errors.New("cannot have nil EpochState")
	errNilStorageState       = errors.New("storage state is nil")
	errNilParentHeader       = errors.New("parent header is nil")
	errInvalidResult         = errors.New("invalid error value")
	errNoEpochData           = errors.New("no epoch data found for upcoming epoch")
	errFirstBlockTimeout     = errors.New("timed out waiting for first block")
	errChannelClosed         = errors.New("block notifier channel was closed")

	other         Other
	invalidCustom InvalidCustom
	unknownCustom UnknownCustom

	dispatchError = scale.MustNewVaryingDataType(other, CannotLookup{}, BadOrigin{}, Module{})
	invalid       = scale.MustNewVaryingDataType(Call{}, Payment{}, Future{}, Stale{}, BadProof{}, AncientBirthBlock{},
		ExhaustsResources{}, invalidCustom, BadMandatory{}, MandatoryDispatch{})
	unknown = scale.MustNewVaryingDataType(ValidityCannotLookup{}, NoUnsignedValidator{}, unknownCustom)

	okRes  = scale.NewResult(nil, dispatchError)
	errRes = scale.NewResult(invalid, unknown)
	result = scale.NewResult(okRes, errRes)
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
	errUnexpectedTxCall         = errors.New("call of the transaction is not expected")
	errInvalidPayment           = errors.New("invalid payment")
	errInvalidTransaction       = errors.New("invalid transaction")
	errOutdatedTransaction      = errors.New("outdated transaction")
	errBadProof                 = errors.New("bad proof")
	errAncientBirthBlock        = errors.New("ancient birth block")
	errExhaustsResources        = errors.New("exhausts resources")
	errMandatoryDispatchError   = errors.New("mandatory dispatch error")
	errInvalidMandatoryDispatch = errors.New("invalid mandatory dispatch")
	errLookupFailed             = errors.New("lookup failed")
	errValidatorNotFound        = errors.New("validator not found")
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

// Index Returns VDT index
func (err Other) Index() uint { return 0 }

// CannotLookup Failed to lookup some data
type CannotLookup struct{}

// Index Returns VDT index
func (err CannotLookup) Index() uint { return 1 }

// BadOrigin A bad origin
type BadOrigin struct{}

// Index Returns VDT index
func (err BadOrigin) Index() uint { return 2 }

// Module A custom error in a module
type Module struct {
	Idx     uint8
	Err     uint8
	Message *string
}

// Index Returns VDT index
func (err Module) Index() uint { return 3 }

func (err Module) string() string {
	return fmt.Sprintf("index: %d code: %d message: %x", err.Idx, err.Err, *err.Message)
}

// ValidityCannotLookup Could not lookup some information that is required to validate the transaction
type ValidityCannotLookup struct{}

// Index Returns VDT index
func (err ValidityCannotLookup) Index() uint { return 0 }

// NoUnsignedValidator No validator found for the given unsigned transaction
type NoUnsignedValidator struct{}

// Index Returns VDT index
func (err NoUnsignedValidator) Index() uint { return 1 }

// UnknownCustom Any other custom unknown validity that is not covered
type UnknownCustom uint8

// Index Returns VDT index
func (err UnknownCustom) Index() uint { return 2 }

// Call The call of the transaction is not expected
type Call struct{}

// Index Returns VDT index
func (err Call) Index() uint { return 0 }

// Payment General error to do with the inability to pay some fees (e.g. account balance too low)
type Payment struct{}

// Index Returns VDT index
func (err Payment) Index() uint { return 1 }

// Future General error to do with the transaction not yet being valid (e.g. nonce too high)
type Future struct{}

// Index Returns VDT index
func (err Future) Index() uint { return 2 }

// Stale General error to do with the transaction being outdated (e.g. nonce too low)
type Stale struct{}

// Index Returns VDT index
func (err Stale) Index() uint { return 3 }

// BadProof General error to do with the transaction’s proofs (e.g. signature)
type BadProof struct{}

// Index Returns VDT index
func (err BadProof) Index() uint { return 4 }

// AncientBirthBlock The transaction birth block is ancient
type AncientBirthBlock struct{}

// Index Returns VDT index
func (err AncientBirthBlock) Index() uint { return 5 }

// ExhaustsResources The transaction would exhaust the resources of current block
type ExhaustsResources struct{}

// Index Returns VDT index
func (err ExhaustsResources) Index() uint { return 6 }

// InvalidCustom Any other custom invalid validity that is not covered
type InvalidCustom uint8

// Index Returns VDT index
func (err InvalidCustom) Index() uint { return 7 }

// BadMandatory An extrinsic with a Mandatory dispatch resulted in Error
type BadMandatory struct{}

// Index Returns VDT index
func (err BadMandatory) Index() uint { return 8 }

// MandatoryDispatch A transaction with a mandatory dispatch
type MandatoryDispatch struct{}

// Index Returns VDT index
func (err MandatoryDispatch) Index() uint { return 9 }

func determineErrType(vdt scale.VaryingDataType) error {
	switch val := vdt.Value().(type) {
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
