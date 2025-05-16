package validatorside

import (
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type CollationFetchingResponseValues interface {
	parachaintypes.Collation | parachaintypes.CollationWithParentHeadData
}

// CollationFetchingResponse represents a response sent by collator
type CollationFetchingResponse struct {
	inner any
}

func setCollationFetchingResponse[Value CollationFetchingResponseValues](mvdt *CollationFetchingResponse, value Value) {
	mvdt.inner = value
}

func (mvdt *CollationFetchingResponse) SetValue(value any) (err error) {
	switch value := value.(type) {
	case parachaintypes.Collation:
		setCollationFetchingResponse(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollationFetchingResponse) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case parachaintypes.Collation:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollationFetchingResponse) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollationFetchingResponse) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(parachaintypes.Collation), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}
