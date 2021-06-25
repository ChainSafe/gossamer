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
	errNilRuntime            = errors.New("runtime is nil")
	errInvalidResult         = errors.New("invalid error value")
	errNoEpochData           = errors.New("no epoch data found for upcoming epoch")
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
	msg string // description of error
}

func (e TransactionValidityError) Error() string {
	return fmt.Sprintf("transaction validity error: %s", e.msg)
}

// Dispatch Errors
type Other string

type CannotLookup struct {
	err string
}

type BadOrigin struct {
	err string
}

type Module struct { // add in `scale:"1"` after
	Idx uint8
	Err   uint8
	Message *string
}

// Dispatch Receivers
func (err Other) Index() uint {return 0}
func (err CannotLookup) Index() uint {return 1}
func (err BadOrigin) Index() uint {return 2}
func (err Module) Index() uint {return 3}
func (err Module) String() string {
	return fmt.Sprintf("index: %d code: %d message: %x", err.Idx, err.Err, *err.Message)
}

// Unknown Transaction Errors
type ValidityCannotLookup struct {
	err string
}

type NoUnsignedValidator struct {
	err string
}

type UnknownCustom uint8

// Unknown Transaction Receivers
func (err ValidityCannotLookup) Index() uint {return 0}
func (err NoUnsignedValidator) Index() uint {return 1}
func (err UnknownCustom) Index() uint {return 2}

// Invalid Transaction Errors
type Call struct {
	err string
}

type Payment struct {
	err string
}

type Future struct {
	err string
}

type Stale struct {
	err string
}

type BadProof struct {
	err string
}

type AncientBirthBlock struct {
	err string
}

type ExhaustsResources struct {
	err string
}

type InvalidCustom uint8

type BadMandatory struct {
	err string
}

type MandatoryDispatch struct {
	err string
}

// Invalid Transaction Receivers
func (err Call) Index() uint {return 0}
func (err Payment) Index() uint {return 1}
func (err Future) Index() uint {return 2}
func (err Stale) Index() uint {return 3}
func (err BadProof) Index() uint {return 4}
func (err AncientBirthBlock) Index() uint {return 5}
func (err ExhaustsResources) Index() uint {return 6}
func (err InvalidCustom) Index() uint {return 7}
func (err BadMandatory) Index() uint {return 8}
func (err MandatoryDispatch) Index() uint {return 9}

func determineDispatchErr(res []byte) error { // This works yay!
	var e Other
	vdt := scale.MustNewVaryingDataType(e, CannotLookup{}, BadOrigin{}, Module{})
	err := scale.Unmarshal(res, &vdt)
	if err != nil {
		return errInvalidResult
	}

	switch val := vdt.Value().(type) {
	case Other:
		return &DispatchOutcomeError{fmt.Sprintf("unknown error: %s", val)}
	case CannotLookup:
		return &DispatchOutcomeError{"failed lookup"}
	case BadOrigin:
		return &DispatchOutcomeError{"bad origin"}
	case Module:
		return &DispatchOutcomeError{fmt.Sprintf("custom module error: %s", val.String())}
	}

	return errInvalidResult
}

func determineInvalidTxnErr(res []byte) error {
	var c InvalidCustom
	vdt := scale.MustNewVaryingDataType(Call{}, Payment{}, Future{}, Stale{}, BadProof{}, AncientBirthBlock{},
										ExhaustsResources{}, c, BadMandatory{}, MandatoryDispatch{})
	err := scale.Unmarshal(res, &vdt)
	if err != nil {
		return errInvalidResult
	}

	switch val := vdt.Value().(type) {
	case Call:
		return &TransactionValidityError{"call of the transaction is not expected"}
	case Payment:
		return &TransactionValidityError{"invalid payment"}
	case Future:
		return &TransactionValidityError{"invalid transaction"}
	case Stale:
		return &TransactionValidityError{"outdated transaction"}
	case BadProof:
		return &TransactionValidityError{"bad proof"}
	case AncientBirthBlock:
		return &TransactionValidityError{"ancient birth block"}
	case ExhaustsResources:
		return &TransactionValidityError{"exhausts resources"}
	case InvalidCustom:
		return &TransactionValidityError{fmt.Sprintf("unknown error: %d", val)}
	case BadMandatory:
		return &TransactionValidityError{"mandatory dispatch error"}
	case MandatoryDispatch:
		return &TransactionValidityError{"invalid mandatory dispatch"}
	}

	return errInvalidResult
}

func determineUnknownTxnErr(res []byte) error {
	var c UnknownCustom
	vdt := scale.MustNewVaryingDataType(ValidityCannotLookup{}, NoUnsignedValidator{}, c)
	err := scale.Unmarshal(res, &vdt)
	if err != nil {
		return errInvalidResult
	}

	switch val := vdt.Value().(type) {
	case ValidityCannotLookup:
		return &TransactionValidityError{"lookup failed"}
	case NoUnsignedValidator:
		return &TransactionValidityError{"validator not found"}
	case UnknownCustom:
		return &TransactionValidityError{fmt.Sprintf("unknown error: %d", val)}
	}

	return errInvalidResult
}


func (err CustomModuleError) String() string {
	return fmt.Sprintf("index: %d code: %d message: %p", err.index, err.err, err.message)
}

func determineErr(res []byte) error {
	switch res[0] {
	case 0:
		switch res[1] {
		case 0:
			return nil
		case 1:
			return determineDispatchErr(res[2:])
		default:
			return errInvalidResult
		}
	case 1:
		switch res[1] {
		case 0:
			return determineInvalidTxnErr(res[2:])
		case 1:
			return determineUnknownTxnErr(res[2:])
		}
	}
	return errInvalidResult
}
