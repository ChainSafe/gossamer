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
type ExecutorParams []ExecutorParam

// NewExecutorParams returns a new ExecutorParams varying data type slice
func NewExecutorParams() ExecutorParams {
	return ExecutorParams{}
}

type ExecutorParamValues interface {
	MaxMemoryPages | StackLogicalMax | StackNativeMax | PrecheckingMaxMemory | PvfPrepTimeout | PvfExecTimeout | WasmExtBulkMemory
}

// ExecutorParam represents the various parameters for modifying the semantics of the execution environment.
type ExecutorParam struct {
	inner any
}

func setExecutorParam[Value ExecutorParamValues](mvdt *ExecutorParam, value Value) {
	mvdt.inner = value
}

func (mvdt *ExecutorParam) SetValue(value any) (err error) {
	switch value := value.(type) {
	case MaxMemoryPages:
		setExecutorParam(mvdt, value)
		return

	case StackLogicalMax:
		setExecutorParam(mvdt, value)
		return

	case StackNativeMax:
		setExecutorParam(mvdt, value)
		return

	case PrecheckingMaxMemory:
		setExecutorParam(mvdt, value)
		return

	case PvfPrepTimeout:
		setExecutorParam(mvdt, value)
		return

	case PvfExecTimeout:
		setExecutorParam(mvdt, value)
		return

	case WasmExtBulkMemory:
		setExecutorParam(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt ExecutorParam) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case MaxMemoryPages:
		return 1, mvdt.inner, nil

	case StackLogicalMax:
		return 2, mvdt.inner, nil

	case StackNativeMax:
		return 3, mvdt.inner, nil

	case PrecheckingMaxMemory:
		return 4, mvdt.inner, nil

	case PvfPrepTimeout:
		return 5, mvdt.inner, nil

	case PvfExecTimeout:
		return 6, mvdt.inner, nil

	case WasmExtBulkMemory:
		return 7, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt ExecutorParam) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt ExecutorParam) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return *new(MaxMemoryPages), nil

	case 2:
		return *new(StackLogicalMax), nil

	case 3:
		return *new(StackNativeMax), nil

	case 4:
		return *new(PrecheckingMaxMemory), nil

	case 5:
		return *new(PvfPrepTimeout), nil

	case 6:
		return *new(PvfExecTimeout), nil

	case 7:
		return *new(WasmExtBulkMemory), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewExecutorParam returns a new ExecutorParam varying data type
func NewExecutorParam() ExecutorParam {

	return ExecutorParam{}
}

// New will enable scale to create new instance when needed
func (ExecutorParam) New() ExecutorParam {
	return NewExecutorParam()
}

// MaxMemoryPages represents the maximum number of memory pages (64KiB bytes per page) that the executor can allocate.
type MaxMemoryPages uint32

// StackLogicalMax defines the limit for the logical stack size in Wasm (maximum number of Wasm values on the stack).
type StackLogicalMax uint32

// StackNativeMax represents the limit of the executor's machine stack size in bytes.
type StackNativeMax uint32

// PrecheckingMaxMemory represents the maximum memory allowance for the preparation worker during pre-checking,
// measured in bytes.
type PrecheckingMaxMemory uint64

// PvfPrepTimeout defines the timeouts for PVF preparation in milliseconds.
type PvfPrepTimeout struct {
	PvfPrepTimeoutKind PvfPrepTimeoutKind `scale:"1"`
	Millisec           uint64             `scale:"2"`
}

// PvfExecTimeout represents the timeouts for PVF execution in milliseconds.
type PvfExecTimeout struct {
	PvfExecTimeoutKind PvfExecTimeoutKind `scale:"1"`
	Millisec           uint64             `scale:"2"`
}

// WasmExtBulkMemory enables the WASM bulk memory proposal.
type WasmExtBulkMemory struct{}

type PvfPrepTimeoutKindValues interface {
	Precheck | Lenient
}

// PvfPrepTimeoutKind is an enumeration representing the type discriminator for PVF preparation timeouts
type PvfPrepTimeoutKind struct {
	inner any
}

func setPvfPrepTimeoutKind[Value PvfPrepTimeoutKindValues](mvdt *PvfPrepTimeoutKind, value Value) {
	mvdt.inner = value
}

func (mvdt *PvfPrepTimeoutKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Precheck:
		setPvfPrepTimeoutKind(mvdt, value)
		return

	case Lenient:
		setPvfPrepTimeoutKind(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt PvfPrepTimeoutKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Precheck:
		return 0, mvdt.inner, nil

	case Lenient:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt PvfPrepTimeoutKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt PvfPrepTimeoutKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Precheck), nil

	case 1:
		return *new(Lenient), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewPvfPrepTimeoutKind returns a new PvfPrepTimeoutKind varying data type
func NewPvfPrepTimeoutKind() PvfPrepTimeoutKind {
	return PvfPrepTimeoutKind{}
}

// Precheck defines the time period for prechecking requests. After this duration,
// an unresponsive preparation worker is considered and will be terminated.
type Precheck struct{}

// Lenient refers to the time period for execution and heads-up requests. It is the duration
// after which the preparation worker is deemed unresponsive and terminated. This timeout
// is more forgiving than the prechecking timeout to avoid honest validators timing out on valid PVFs.
type Lenient struct{}

type PvfExecTimeoutKindValues interface {
	Backing | Approval
}

// PvfExecTimeoutKind is an enumeration representing the type discriminator for PVF execution timeouts
type PvfExecTimeoutKind struct {
	inner any
}

func setPvfExecTimeoutKind[Value PvfExecTimeoutKindValues](mvdt *PvfExecTimeoutKind, value Value) {
	mvdt.inner = value
}

func (mvdt *PvfExecTimeoutKind) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Backing:
		setPvfExecTimeoutKind(mvdt, value)
		return

	case Approval:
		setPvfExecTimeoutKind(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt PvfExecTimeoutKind) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Backing:
		return 0, mvdt.inner, nil

	case Approval:
		return 1, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt PvfExecTimeoutKind) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt PvfExecTimeoutKind) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Backing), nil

	case 1:
		return *new(Approval), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewPvfExecTimeoutKind returns a new PvfExecTimeoutKind varying data type
func NewPvfExecTimeoutKind() PvfExecTimeoutKind {
	return PvfExecTimeoutKind{}
}

// Backing represents the amount of time to spend on execution during backing.
type Backing struct{}

// Approval represents the amount of time to spend on execution during approval or disputes.
// This timeout should be much longer than the backing execution timeout to ensure that,
// in the absence of extremely large disparities between hardware, blocks that pass
// backing are considered executable by approval checkers or dispute participants.
type Approval struct{}
