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
	// ErrAuthIndexOutOfBound is returned when a authority index doesn't exist
	ErrAuthIndexOutOfBound = errors.New("authority index doesn't exist")

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

	// ErrThresholdOneIsZero is returned when one of or both parameters to CalculateThreshold is zero
	ErrThresholdOneIsZero = errors.New("numerator or denominator cannot be 0")

	errNilParentHeader            = errors.New("parent header is nil")
	errInvalidResult              = errors.New("invalid error value")
	errOverPrimarySlotThreshold   = errors.New("cannot claim slot, over primary threshold")
	errNotOurTurnToPropose        = errors.New("cannot claim slot, not our turn to propose a block")
	errMissingDigestItems         = errors.New("block header is missing digest items")
	errServicePaused              = errors.New("service paused")
	errInvalidSlotTechnique       = errors.New("invalid slot claiming technique")
	errNoBABEAuthorityKeyProvided = errors.New("cannot create BABE service as authority; no keypair provided")
	errLastDigestItemNotSeal      = errors.New("last digest item is not seal")
	errLaggingSlot                = errors.New("current slot is smaller than slot of best block")
	errNoDigest                   = errors.New("no digest provided")

	// other         Other
	// invalidCustom InvalidCustom
	// unknownCustom UnknownCustom
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
	errBadSigner                = errors.New("invalid signing address")
)

func newUnknownError(data any) error {
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

// // Index returns VDT index
// func (Other) Index() uint { return 0 }

func (o Other) String() string { return string(o) }

// CannotLookup Failed to lookup some data
type CannotLookup struct{}

// // Index returns VDT index
// func (CannotLookup) Index() uint { return 1 }

func (CannotLookup) String() string { return "cannot lookup" }

// BadOrigin A bad origin
type BadOrigin struct{}

// // Index returns VDT index
// func (BadOrigin) Index() uint { return 2 }

func (BadOrigin) String() string { return "bad origin" }

// Module A custom error in a module
type Module struct {
	Idx     uint8
	Err     uint8
	Message *string
}

// // Index returns VDT index
// func (Module) Index() uint { return 3 }

func (err Module) String() string {
	message := "nil"
	if err.Message != nil {
		message = *err.Message
	}
	return fmt.Sprintf("Module{Idx=%d, Err=%d Message=%s", err.Idx, err.Err, message)
}

// ValidityCannotLookup Could not lookup some information that is required to validate the transaction
type ValidityCannotLookup struct{}

// // Index returns VDT index
// func (ValidityCannotLookup) Index() uint { return 0 }

func (ValidityCannotLookup) String() string { return "validity cannot lookup" }

// NoUnsignedValidator No validator found for the given unsigned transaction
type NoUnsignedValidator struct{}

// // Index returns VDT index
// func (NoUnsignedValidator) Index() uint { return 1 }

func (NoUnsignedValidator) String() string { return "no unsigned validator" }

// UnknownCustom Any other custom unknown validity that is not covered
type UnknownCustom uint8

// // Index returns VDT index
// func (UnknownCustom) Index() uint { return 2 }

func (uc UnknownCustom) String() string { return fmt.Sprintf("UnknownCustom(%d)", uc) }

// Call The call of the transaction is not expected
type Call struct{}

// // Index returns VDT index
// func (Call) Index() uint { return 0 }

func (Call) String() string { return "call" }

// Payment General error to do with the inability to pay some fees (e.g. account balance too low)
type Payment struct{}

// // Index returns VDT index
// func (Payment) Index() uint { return 1 }

func (Payment) String() string { return "payment" }

// Future General error to do with the transaction not yet being valid (e.g. nonce too high)
type Future struct{}

// // Index returns VDT index
// func (Future) Index() uint { return 2 }

func (Future) String() string { return "future" }

// Stale General error to do with the transaction being outdated (e.g. nonce too low)
type Stale struct{}

// // Index returns VDT index
// func (Stale) Index() uint { return 3 }

func (Stale) String() string { return "stale" }

// BadProof General error to do with the transaction’s proofs (e.g. signature)
type BadProof struct{}

// // Index returns VDT index
// func (BadProof) Index() uint { return 4 }

func (BadProof) String() string { return "bad proof" }

// AncientBirthBlock The transaction birth block is ancient
type AncientBirthBlock struct{}

// // Index returns VDT index
// func (AncientBirthBlock) Index() uint { return 5 }

func (AncientBirthBlock) String() string { return "ancient birth block" }

// ExhaustsResources The transaction would exhaust the resources of current block
type ExhaustsResources struct{}

// // Index returns VDT index
// func (ExhaustsResources) Index() uint { return 6 }

func (ExhaustsResources) String() string { return "exhausts resources" }

// InvalidCustom Any other custom invalid validity that is not covered
type InvalidCustom uint8

// // Index returns VDT index
// func (InvalidCustom) Index() uint { return 7 }

func (ic InvalidCustom) String() string { return fmt.Sprintf("InvalidCustom(%d)", ic) }

// BadMandatory An extrinsic with a Mandatory dispatch resulted in Error
type BadMandatory struct{}

// // Index returns VDT index
// func (BadMandatory) Index() uint { return 8 }

func (BadMandatory) String() string { return "bad mandatory" }

// MandatoryDispatch A transaction with a mandatory dispatch
type MandatoryDispatch struct{}

// // Index returns VDT index
// func (MandatoryDispatch) Index() uint { return 9 }

func (MandatoryDispatch) String() string { return "mandatory dispatch" }

// BadSigner A transaction with a mandatory dispatch
type BadSigner struct{}

// // Index returns VDT index
// func (BadSigner) Index() uint { return 10 }

func (BadSigner) String() string { return "invalid signing address" }

func determineErrType(vdt scale.EncodeVaryingDataType) (err error) {
	vdtVal, err := vdt.Value()
	if err != nil {
		return fmt.Errorf("getting vdt value: %w", err)
	}

	switch val := vdtVal.(type) {
	case Other:
		err = &DispatchOutcomeError{fmt.Sprintf("unknown error: %s", val)}
	case CannotLookup:
		err = &DispatchOutcomeError{"failed lookup"}
	case BadOrigin:
		err = &DispatchOutcomeError{"bad origin"}
	case Module:
		err = &DispatchOutcomeError{fmt.Sprintf("custom module error: %s", val)}
	case Call:
		err = &TransactionValidityError{errUnexpectedTxCall}
	case Payment:
		err = &TransactionValidityError{errInvalidPayment}
	case Future:
		err = &TransactionValidityError{errInvalidTransaction}
	case Stale:
		err = &TransactionValidityError{errOutdatedTransaction}
	case BadProof:
		err = &TransactionValidityError{errBadProof}
	case AncientBirthBlock:
		err = &TransactionValidityError{errAncientBirthBlock}
	case ExhaustsResources:
		err = &TransactionValidityError{errExhaustsResources}
	case InvalidCustom:
		err = &TransactionValidityError{newUnknownError(val)}
	case BadMandatory:
		err = &TransactionValidityError{errMandatoryDispatchError}
	case MandatoryDispatch:
		err = &TransactionValidityError{errInvalidMandatoryDispatch}
	case ValidityCannotLookup:
		err = &TransactionValidityError{errLookupFailed}
	case NoUnsignedValidator:
		err = &TransactionValidityError{errValidatorNotFound}
	case UnknownCustom:
		err = &TransactionValidityError{newUnknownError(val)}
	case BadSigner:
		err = &TransactionValidityError{errBadSigner}
	default:
		err = errInvalidResult
	}

	return err
}

type dispatchErrorValues interface {
	Other | CannotLookup | BadOrigin | Module
}
type dispatchError struct {
	inner any
}

func setdispatchError[Value dispatchErrorValues](mvdt *dispatchError, value Value) {
	mvdt.inner = value
}

func (mvdt *dispatchError) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Other:
		setdispatchError(mvdt, value)
		return
	case CannotLookup:
		setdispatchError(mvdt, value)
		return
	case BadOrigin:
		setdispatchError(mvdt, value)
		return
	case Module:
		setdispatchError(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt dispatchError) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Other:
		return 0, mvdt.inner, nil
	case CannotLookup:
		return 1, mvdt.inner, nil
	case BadOrigin:
		return 2, mvdt.inner, nil
	case Module:
		return 3, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt dispatchError) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt dispatchError) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Other), nil
	case 1:
		return *new(CannotLookup), nil
	case 2:
		return *new(BadOrigin), nil
	case 3:
		return *new(Module), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

type invalidValues interface {
	Call | Payment | Future | Stale | BadProof | AncientBirthBlock |
		ExhaustsResources | InvalidCustom | BadMandatory | MandatoryDispatch | BadSigner
}
type invalid struct {
	inner any
}

func setinvalid[Value invalidValues](mvdt *invalid, value Value) {
	mvdt.inner = value
}

func (mvdt *invalid) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Call:
		setinvalid(mvdt, value)
		return
	case Payment:
		setinvalid(mvdt, value)
		return
	case Future:
		setinvalid(mvdt, value)
		return
	case Stale:
		setinvalid(mvdt, value)
		return
	case BadProof:
		setinvalid(mvdt, value)
		return
	case AncientBirthBlock:
		setinvalid(mvdt, value)
		return
	case ExhaustsResources:
		setinvalid(mvdt, value)
		return
	case InvalidCustom:
		setinvalid(mvdt, value)
		return
	case BadMandatory:
		setinvalid(mvdt, value)
		return
	case MandatoryDispatch:
		setinvalid(mvdt, value)
		return
	case BadSigner:
		setinvalid(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt invalid) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Call:
		return 0, mvdt.inner, nil
	case Payment:
		return 1, mvdt.inner, nil
	case Future:
		return 2, mvdt.inner, nil
	case Stale:
		return 3, mvdt.inner, nil
	case BadProof:
		return 4, mvdt.inner, nil
	case AncientBirthBlock:
		return 5, mvdt.inner, nil
	case ExhaustsResources:
		return 6, mvdt.inner, nil
	case InvalidCustom:
		return 7, mvdt.inner, nil
	case BadMandatory:
		return 8, mvdt.inner, nil
	case MandatoryDispatch:
		return 9, mvdt.inner, nil
	case BadSigner:
		return 10, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt invalid) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}
func (mvdt invalid) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Call), nil
	case 1:
		return *new(Payment), nil
	case 2:
		return *new(Future), nil
	case 3:
		return *new(Stale), nil
	case 4:
		return *new(BadProof), nil
	case 5:
		return *new(AncientBirthBlock), nil
	case 6:
		return *new(ExhaustsResources), nil
	case 7:
		return *new(InvalidCustom), nil
	case 8:
		return *new(BadMandatory), nil
	case 9:
		return *new(MandatoryDispatch), nil
	case 10:
		return *new(BadSigner), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

type unknownValues interface {
	ValidityCannotLookup | NoUnsignedValidator | UnknownCustom
}
type unknown struct {
	inner any
}

func setunknown[Value unknownValues](mvdt *unknown, value Value) {
	mvdt.inner = value
}

func (mvdt *unknown) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ValidityCannotLookup:
		setunknown(mvdt, value)
		return
	case NoUnsignedValidator:
		setunknown(mvdt, value)
		return
	case UnknownCustom:
		setunknown(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt unknown) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ValidityCannotLookup:
		return 0, mvdt.inner, nil
	case NoUnsignedValidator:
		return 1, mvdt.inner, nil
	case UnknownCustom:
		return 2, mvdt.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt unknown) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt unknown) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(ValidityCannotLookup), nil
	case 1:
		return *new(NoUnsignedValidator), nil
	case 2:
		return *new(UnknownCustom), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

func determineErr(res []byte) error {
	// dispatchError := scale.MustNewVaryingDataType(other, CannotLookup{}, BadOrigin{}, Module{})
	// invalid := scale.MustNewVaryingDataType(Call{}, Payment{}, Future{}, Stale{}, BadProof{}, AncientBirthBlock{},
	// 	ExhaustsResources{}, invalidCustom, BadMandatory{}, MandatoryDispatch{}, BadSigner{})
	// unknown := scale.MustNewVaryingDataType(ValidityCannotLookup{}, NoUnsignedValidator{}, unknownCustom)
	dispatchError := dispatchError{}
	invalid := invalid{}
	unknown := unknown{}

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
					return determineErrType(err.Err.(scale.EncodeVaryingDataType))
				default:
					return errInvalidResult
				}
			} else {
				return determineErrType(ok.(scale.EncodeVaryingDataType))
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
					return determineErrType(err.Err.(scale.EncodeVaryingDataType))
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
