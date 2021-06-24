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
	"github.com/ChainSafe/gossamer/lib/common/optional"
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
)

// A DispatchOutcomeError is outcome of dispatching the extrinsic
type DispatchOutcomeError struct {
	msg string // description of error
}

//type DispatchOutcomeError string

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

func determineCustomModuleErr(res []byte) error {
	if len(res) < 3 {
		return errInvalidResult
	}
	errMsg, err := optional.NewBytes(false, nil).DecodeBytes(res[2:])
	if err != nil {
		return err
	}
	return fmt.Errorf("index: %d code: %d message: %s", res[0], res[1], errMsg.String())
}

//func determineDispatchErr(res []byte) error {
//	var v []byte
//	err := scale.Unmarshal(res[1:], &v)
//	if err != nil {
//		// this err is thrown if we can't unmarshal, so prolly `errInvalidResult`. Check to make sure
//		// TODO Create stucts for errors and integrate into Varying data type
//		return errInvalidResult
//	}
//
//	switch res[0] {
//	case 0:
//		var v []byte
//		err := scale.Unmarshal(res[1:], &v)
//		if err != nil {
//			// this err is thrown if we can't unmarshal, so prolly `errInvalidResult`. Check to make sure
//			// TODO Create stucts for errors and integrate into Varying data type
//			return errInvalidResult
//		}
//		return &DispatchOutcomeError{fmt.Sprintf("unknown error: %s", v)}
//	case 1:
//		return &DispatchOutcomeError{"failed lookup"}
//	case 2:
//		return &DispatchOutcomeError{"bad origin"}
//	case 3:
//		return &DispatchOutcomeError{fmt.Sprintf("custom module error: %s", determineCustomModuleErr(res[1:]))}
//	}
//
//	return errInvalidResult
//}

/*
	Two main issues I need to fix:
		1) Types arent being unmarshalled correctly: probably because of how I constructed them or how I am unmarshalling - fixed
		2) CustomModuleError data isnt being decoded. The type is but the struct is empty
 */
func determineDispatchErr(res []byte) error {
	// Maybe I need to do something with this first status byte? unsure of what tho
	// If not encoded with thees types, will they still evaluate? Maybe thats why they are going to customModuleError
	var e UnknownError
	vdt := scale.MustNewVaryingDataType(e, FailedLookup{}, BadOrigin{}, CustomModuleError{})
	// Am I unmarshalling the right thing here? Make sure should be res[1]
	err := scale.Unmarshal(res, &vdt)
	if err != nil {
		// Unmarshalling error. Check to make sure this is the correct return type for this case
		return errInvalidResult
	}

	fmt.Println("Vdt")
	fmt.Println(vdt.Value())

	// Might have to change testing to adjust to new types
	// Something is wrong with my types: Not being properly recognized
	switch val := vdt.Value().(type){
	case UnknownError: // Got it!
		return &DispatchOutcomeError{fmt.Sprintf("unknown error: %s", val)}
	case FailedLookup:
		 // Add testing for this case, make sure struct is correct
		fmt.Println("failed lookup")
		return &DispatchOutcomeError{"failed lookup"}
	case BadOrigin:
		// Add testing for this case, make sure struct is correct
		fmt.Println("bad origin")
		return &DispatchOutcomeError{"bad origin"}
	case CustomModuleError:
		// Printing nice, not getting the correct values
		fmt.Println("Val2:")
		fmt.Println(val)
		return &DispatchOutcomeError{fmt.Sprintf("custom module error: %s", val.String())}
	default: // Remove this before PR lol
		fmt.Println("Something is most definitly afoot at the circle K")
	}

	return errInvalidResult
}

func determineInvalidTxnErr(res []byte) error {
	switch res[0] {
	case 0:
		return &TransactionValidityError{"call of the transaction is not expected"}
	case 1:
		return &TransactionValidityError{"invalid payment"}
	case 2:
		return &TransactionValidityError{"invalid transaction"}
	case 3:
		return &TransactionValidityError{"outdated transaction"}
	case 4:
		return &TransactionValidityError{"bad proof"}
	case 5:
		return &TransactionValidityError{"ancient birth block"}
	case 6:
		return &TransactionValidityError{"exhausts resources"}
	case 7:
		return &TransactionValidityError{fmt.Sprintf("unknown error: %d", res[1])}
	case 8:
		return &TransactionValidityError{"mandatory dispatch error"}
	case 9:
		return &TransactionValidityError{"invalid mandatory dispatch"}
	}
	return errInvalidResult
}

func determineUnknownTxnErr(res []byte) error {
	switch res[0] {
	case 0:
		return &TransactionValidityError{"lookup failed"}
	case 1:
		return &TransactionValidityError{"validator not found"}
	case 2:
		return &TransactionValidityError{fmt.Sprintf("unknown error: %d", res[1])}
	}
	return errInvalidResult
}

type UnknownError string

func (err UnknownError) Index() uint {
	return 0
}

type FailedLookup struct {
	err string
}

func (err FailedLookup) Index() uint {
	return 1
}

type BadOrigin struct {
	err string
}

func (err BadOrigin) Index() uint {
	return 2
}

type CustomModuleError struct {
	index uint8       `scale:"3"`
	err   uint8	  	  `scale:"2"`
	message *string   `scale:"1"` // Does scale need to consider that rust has multiple string types?
}

func (err CustomModuleError) Index() uint {
	return 3
}

func (err CustomModuleError) String() string {
	return fmt.Sprintf("index: %d code: %d message: %p", err.index, err.err, err.message)
}

func determineErr(res []byte) error {
	switch res[0] {
	case 0: // DispatchOutcome
		switch res[1] {
		case 0:
			return nil
		case 1:
			return determineDispatchErr(res[2:])
		default:
			return errInvalidResult
		}
	case 1: // TransactionValidityError
		switch res[1] {
		case 0:
			return determineInvalidTxnErr(res[2:])
		case 1:
			return determineUnknownTxnErr(res[2:])
		}
	}
	return errInvalidResult
}
