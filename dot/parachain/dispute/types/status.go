package types

import (
	"fmt"
	"github.com/pkg/errors"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

var ErrDisputeNotConcluded = errors.New("dispute not concluded")

// ActiveStatus is the status when the dispute is active.
type ActiveStatus struct{}

// Index returns the index of the type ActiveStatus.
func (ActiveStatus) Index() uint {
	return 0
}

// ConcludedForStatus is the status when the dispute is concluded for the candidate.
type ConcludedForStatus struct {
	Since uint64
}

// Index returns the index of the type ConcludedForStatus.
func (ConcludedForStatus) Index() uint {
	return 1
}

// ConcludedAgainstStatus is the status when the dispute is concluded against the candidate.
type ConcludedAgainstStatus struct {
	Since uint64
}

// Index returns the index of the type ConcludedAgainstStatus.
func (ConcludedAgainstStatus) Index() uint {
	return 2
}

// ConfirmedStatus is the status when the dispute is confirmed.
type ConfirmedStatus struct{}

// Index returns the index of the type ConfirmedStatus.
func (ConfirmedStatus) Index() uint {
	return 3
}

// DisputeStatus is the status of a dispute.
type DisputeStatus scale.VaryingDataType

// Set will set a VaryingDataTypeValue using the underlying VaryingDataType
func (ds *DisputeStatus) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*ds)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}
	*ds = DisputeStatus(vdt)
	return nil
}

// Value returns the value from the underlying VaryingDataType
func (ds *DisputeStatus) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*ds)
	return vdt.Value()
}

// Confirm confirms the dispute, if not concluded/confirmed already.
func (ds *DisputeStatus) Confirm() error {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	switch val.(type) {
	case ActiveStatus:
		return ds.Set(ConfirmedStatus{})
	default:
		return nil
	}
}

// ConcludeFor transitions the status to a new status where the dispute is concluded for the candidate.
func (ds *DisputeStatus) ConcludeFor(since uint64) error {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	switch status := val.(type) {
	case ActiveStatus, ConfirmedStatus:
		return ds.Set(ConcludedForStatus{Since: since})
	case ConcludedForStatus:
		if since < status.Since {
			if err := ds.Set(ConcludedForStatus{Since: since}); err != nil {
				return fmt.Errorf("setting dispute status to ConcludedFor: %w", err)
			}
		}
	default:
		return fmt.Errorf("invalid dispute status type %T", val)
	}

	return nil
}

// ConcludeAgainst transitions the status to a new status where the dispute is concluded against the candidate.
func (ds *DisputeStatus) ConcludeAgainst(since uint64) error {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	switch status := val.(type) {
	case ActiveStatus, ConfirmedStatus:
		return ds.Set(ConcludedAgainstStatus{Since: since})
	case ConcludedForStatus:
		if since < status.Since {
			if err := ds.Set(ConcludedAgainstStatus{Since: since}); err != nil {
				return fmt.Errorf("setting dispute status to ConcludedAgainst: %w", err)
			}
		}
	case ConcludedAgainstStatus:
		if since < status.Since {
			if err := ds.Set(ConcludedAgainstStatus{Since: since}); err != nil {
				return fmt.Errorf("setting dispute status to ConcludedAgainst: %w", err)
			}
		}
	default:
		return fmt.Errorf("invalid dispute status type %T", val)
	}

	return nil
}

// ConcludedAt returns the time the dispute was concluded, if it is concluded.
func (ds *DisputeStatus) ConcludedAt() (*uint64, error) {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return nil, fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	switch status := val.(type) {
	case ActiveStatus, ConfirmedStatus:
		return nil, ErrDisputeNotConcluded
	case ConcludedForStatus:
		return &status.Since, nil
	case ConcludedAgainstStatus:
		return &status.Since, nil
	default:
		return nil, fmt.Errorf("invalid dispute status type")
	}
}

// IsConfirmedConcluded returns true if the dispute is confirmed or concluded.
func (ds *DisputeStatus) IsConfirmedConcluded() (bool, error) {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	switch val.(type) {
	case ConfirmedStatus, ConcludedForStatus, ConcludedAgainstStatus:
		return true, nil
	default:
		return false, nil
	}
}

// IsConfirmed returns true if the dispute is confirmed.
func (ds *DisputeStatus) IsConfirmed() (bool, error) {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	if _, ok := val.(ConfirmedStatus); ok {
		return true, nil
	}

	return false, nil
}

// IsConcludedFor returns true if the dispute is concluded for the candidate.
func (ds *DisputeStatus) IsConcludedFor() (bool, error) {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	if _, ok := val.(ConcludedForStatus); ok {
		return true, nil
	}

	return false, nil
}

// IsConcludedAgainst returns true if the dispute is concluded against the candidate.
func (ds *DisputeStatus) IsConcludedAgainst() (bool, error) {
	vdt := scale.VaryingDataType(*ds)
	val, err := vdt.Value()
	if err != nil {
		return false, fmt.Errorf("getting value from DisputeStatus vdt: %w", err)
	}

	if _, ok := val.(ConcludedAgainstStatus); ok {
		return true, nil
	}

	return false, nil
}

// NewDisputeStatus returns a new DisputeStatus.
func NewDisputeStatus() (DisputeStatus, error) {
	vdt, err := scale.NewVaryingDataType(ActiveStatus{},
		ConcludedForStatus{},
		ConcludedAgainstStatus{},
		ConfirmedStatus{},
	)
	if err != nil {
		return DisputeStatus{}, fmt.Errorf("creating new dispute status vdt: %w", err)
	}

	return DisputeStatus(vdt), nil
}
