// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachaintypes

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// ExecutorParams represents the abstract semantics of an execution environment and should remain
// as abstract as possible. There are no mandatory parameters defined at the moment, and if any
// are introduced in the future, they must be clearly documented as mandatory.
type ExecutorParams scale.VaryingDataTypeSlice

// NewExecutorParams returns a new ExecutorParams varying data type slice
func NewExecutorParams() ExecutorParams {
	vdt := NewExecutorParam()
	vdts := scale.NewVaryingDataTypeSlice(scale.VaryingDataType(vdt))
	return ExecutorParams(vdts)
}

// Add takes variadic parameter values to add VaryingDataTypeValue
func (e *ExecutorParams) Add(val scale.VaryingDataTypeValue) (err error) {
	slice := scale.VaryingDataTypeSlice(*e)
	err = slice.Add(val)
	if err != nil {
		return fmt.Errorf("adding value to varying data type slice: %w", err)
	}

	*e = ExecutorParams(slice)
	return nil
}

// ExecutorParam represents the various parameters for modifying the semantics of the execution environment.
type ExecutorParam scale.VaryingDataType

// NewExecutorParam returns a new ExecutorParam varying data type
func NewExecutorParam() ExecutorParam {
	vdt := scale.MustNewVaryingDataType(
		MaxMemoryPages(0),
		StackLogicalMax(0),
		StackNativeMax(0),
		PrecheckingMaxMemory(0),
		PvfPrepTimeout{},
		PvfExecTimeout{},
		WasmExtBulkMemory{},
	)
	return ExecutorParam(vdt)
}

// New will enable scale to create new instance when needed
func (ExecutorParam) New() ExecutorParam {
	return NewExecutorParam()
}

// Set will set a value using the underlying  varying data type
func (s *ExecutorParam) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = ExecutorParam(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *ExecutorParam) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// MaxMemoryPages represents the maximum number of memory pages (64KiB bytes per page) that the executor can allocate.
type MaxMemoryPages uint32

// Index returns the index of varying data type
func (MaxMemoryPages) Index() uint {
	return 1
}

// StackLogicalMax defines the limit for the logical stack size in Wasm (maximum number of Wasm values on the stack).
type StackLogicalMax uint32

// Index returns the index of varying data type
func (StackLogicalMax) Index() uint {
	return 2
}

// StackNativeMax represents the limit of the executor's machine stack size in bytes.
type StackNativeMax uint32

// Index returns the index of varying data type
func (StackNativeMax) Index() uint {
	return 3
}

// PrecheckingMaxMemory represents the maximum memory allowance for the preparation worker during pre-checking,
// measured in bytes.
type PrecheckingMaxMemory uint64

// Index returns the index of varying data type
func (PrecheckingMaxMemory) Index() uint {
	return 4
}

// PvfPrepTimeout defines the timeouts for PVF preparation in milliseconds.
type PvfPrepTimeout struct {
	PvfPrepTimeoutKind PvfPrepTimeoutKind `scale:"1"`
	Millisec           uint64             `scale:"2"`
}

// Index returns the index of varying data type
func (PvfPrepTimeout) Index() uint {
	return 5
}

// PvfExecTimeout represents the timeouts for PVF execution in milliseconds.
type PvfExecTimeout struct {
	PvfExecTimeoutKind PvfExecTimeoutKind `scale:"1"`
	Millisec           uint64             `scale:"2"`
}

// Index returns the index of varying data type
func (PvfExecTimeout) Index() uint {
	return 6
}

// WasmExtBulkMemory enables the WASM bulk memory proposal.
type WasmExtBulkMemory struct{}

// Index returns the index of varying data type
func (WasmExtBulkMemory) Index() uint {
	return 7
}

// PvfPrepTimeoutKind is an enumeration representing the type discriminator for PVF preparation timeouts
type PvfPrepTimeoutKind scale.VaryingDataType

// NewPvfPrepTimeoutKind returns a new PvfPrepTimeoutKind varying data type
func NewPvfPrepTimeoutKind() PvfPrepTimeoutKind {
	vdt := scale.MustNewVaryingDataType(Precheck{}, Lenient{})
	return PvfPrepTimeoutKind(vdt)
}

// New will enable scale to create new instance when needed
func (PvfPrepTimeoutKind) New() PvfPrepTimeoutKind {
	return NewPvfPrepTimeoutKind()
}

// Set will set a value using the underlying  varying data type
func (p *PvfPrepTimeoutKind) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*p)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*p = PvfPrepTimeoutKind(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *PvfPrepTimeoutKind) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// Precheck defines the time period for prechecking requests. After this duration,
// an unresponsive preparation worker is considered and will be terminated.
type Precheck struct{}

// Index returns the index of varying data type
func (Precheck) Index() uint {
	return 0
}

// Lenient refers to the time period for execution and heads-up requests. It is the duration
// after which the preparation worker is deemed unresponsive and terminated. This timeout
// is more forgiving than the prechecking timeout to avoid honest validators timing out on valid PVFs.
type Lenient struct{}

// Index returns the index of varying data type
func (Lenient) Index() uint {
	return 1
}

// PvfExecTimeoutKind is an enumeration representing the type discriminator for PVF execution timeouts
type PvfExecTimeoutKind scale.VaryingDataType

// NewPvfExecTimeoutKind returns a new PvfExecTimeoutKind varying data type
func NewPvfExecTimeoutKind() PvfExecTimeoutKind {
	vdt := scale.MustNewVaryingDataType(Backing{}, Approval{})
	return PvfExecTimeoutKind(vdt)
}

// New will enable scale to create new instance when needed
func (PvfExecTimeoutKind) New() PvfExecTimeoutKind {
	return NewPvfExecTimeoutKind()
}

// Set will set a value using the underlying  varying data type
func (s *PvfExecTimeoutKind) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = PvfExecTimeoutKind(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *PvfExecTimeoutKind) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// Backing represents the amount of time to spend on execution during backing.
type Backing struct{}

// Index returns the index of varying data type
func (Backing) Index() uint {
	return 0
}

// Approval represents the amount of time to spend on execution during approval or disputes.
// This timeout should be much longer than the backing execution timeout to ensure that,
// in the absence of extremely large disparities between hardware, blocks that pass
// backing are considered executable by approval checkers or dispute participants.
type Approval struct{}

// Index returns the index of varying data type
func (Approval) Index() uint {
	return 1
}
