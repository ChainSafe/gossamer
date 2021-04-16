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
	"github.com/ChainSafe/gossamer/lib/scale"
)

// ErrBadSlotClaim is returned when a slot claim is invalid
var ErrBadSlotClaim = errors.New("could not verify slot claim VRF proof")

// ErrBadSecondarySlotClaim is returned when a slot claim is invalid
var ErrBadSecondarySlotClaim = errors.New("invalid secondary slot claim")

// ErrBadSignature is returned when a seal is invalid
var ErrBadSignature = errors.New("could not verify signature")

// ErrProducerEquivocated is returned when a block producer has produced conflicting blocks
var ErrProducerEquivocated = errors.New("block producer equivocated")

// ErrNilBlockState is returned when the BlockState is nil
var ErrNilBlockState = errors.New("cannot have nil BlockState")

// ErrNilEpochState is returned when the EpochState is nil
var ErrNilEpochState = errors.New("cannot have nil EpochState")

// ErrNotAuthorized is returned when the node is not authorized to produce a block
var ErrNotAuthorized = errors.New("not authorized to produce block")

// ErrNoBABEHeader is returned when there is no BABE header found for a block, specifically when calculating randomness
var ErrNoBABEHeader = errors.New("no BABE header found for block")

// ErrVRFOutputOverThreshold is returned when the vrf output for a block is invalid
var ErrVRFOutputOverThreshold = errors.New("vrf output over threshold")

// ErrInvalidBlockProducerIndex is returned when the producer of a block isn't in the authority set
var ErrInvalidBlockProducerIndex = errors.New("block producer is not in authority set")

// ErrAuthorityAlreadyDisabled is returned when attempting to disabled an already-disabled authority
var ErrAuthorityAlreadyDisabled = errors.New("authority has already been disabled")

// ErrAuthorityDisabled is returned when attempting to verify a block produced by a disabled authority
var ErrAuthorityDisabled = errors.New("authority has been disabled for the remaining slots in the epoch")

// ErrNotAuthority is returned when trying to perform authority functions when not an authority
var ErrNotAuthority = errors.New("node is not an authority")

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

func determineDispatchErr(res []byte) error {
	switch res[0] {
	case 0:
		unKnownError, _ := scale.Decode(res[1:], []byte{})
		return fmt.Errorf("unknown error: %s", string(unKnownError.([]byte)))
	case 1:
		return fmt.Errorf("failed lookup")
	case 2:
		return fmt.Errorf("bad origin")
	case 3:
		return fmt.Errorf("custom module error: %s", determineCustomModuleErr(res[1:]))
	}
	return errInvalidResult
}

func determineInvalidTxnErr(res []byte) error {
	switch res[0] {
	case 0:
		return fmt.Errorf("call of the transaction is not expected")
	case 1:
		return fmt.Errorf("invalid payment")
	case 2:
		return fmt.Errorf("invalid transaction")
	case 3:
		return fmt.Errorf("outdated transaction")
	case 4:
		return fmt.Errorf("bad proof")
	case 5:
		return fmt.Errorf("ancient birth block")
	case 6:
		return fmt.Errorf("exhausts resources")
	case 7:
		return fmt.Errorf("unknown error: %d", res[1])
	case 8:
		return fmt.Errorf("mandatory dispatch error")
	case 9:
		return fmt.Errorf("invalid mandatory dispatch")
	}
	return errInvalidResult
}

func determineUnknownTxnErr(res []byte) error {
	switch res[0] {
	case 0:
		return fmt.Errorf("lookup failed")
	case 1:
		return fmt.Errorf("validator not found")
	case 2:
		return fmt.Errorf("unknown error: %d", res[1])
	}
	return errInvalidResult
}

func determineErr(res []byte) (bool, error) {
	switch res[0] {
	case 0: // DispatchOutcome
		switch res[1] {
		case 0:
			return true, nil
		case 1:
			return true, determineDispatchErr(res[2:])
		default:
			return true, errInvalidResult
		}
	case 1: // TransactionValidityError
		switch res[1] {
		case 0:
			return false, determineInvalidTxnErr(res[2:])
		case 1:
			return false, determineUnknownTxnErr(res[2:])
		}
	}
	return false, errInvalidResult
}
