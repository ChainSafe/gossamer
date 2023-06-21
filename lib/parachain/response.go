package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/pkg/scale"
)

// StatementFetchingResponse represents the statement fetching response is
// sent by nodes to the clients who issued a collation fetching request.
//
// Respond with found full statement.
type StatementFetchingResponse scale.VaryingDataType

// MissingDataInStatement represents the data missing to reconstruct the full signed statement.
type MissingDataInStatement CommittedCandidateReceipt

// Index returns the index of varying data type
func (MissingDataInStatement) Index() uint {
	return 0
}

// NewStatementFetchingResponse returns a new statement fetching response varying data type
func NewStatementFetchingResponse() StatementFetchingResponse {
	vdt := scale.MustNewVaryingDataType(MissingDataInStatement{})
	return StatementFetchingResponse(vdt)
}

// Set will set a value using the underlying  varying data type
func (s *StatementFetchingResponse) Set(val scale.VaryingDataTypeValue) (err error) {
	vdt := scale.VaryingDataType(*s)
	err = vdt.Set(val)
	if err != nil {
		return fmt.Errorf("setting value to varying data type: %w", err)
	}

	*s = StatementFetchingResponse(vdt)
	return nil
}

// Value returns the value from the underlying varying data type
func (s *StatementFetchingResponse) Value() (scale.VaryingDataTypeValue, error) {
	vdt := scale.VaryingDataType(*s)
	return vdt.Value()
}

// Encode returns the SCALE encoding of the StatementFetchingResponse.
func (s *StatementFetchingResponse) Encode() ([]byte, error) {
	return scale.Marshal(*s)
}

// Decode returns the SCALE decoding of the StatementFetchingResponse.
func (s *StatementFetchingResponse) Decode(in []byte) (err error) {
	return scale.Unmarshal(in, s)
}

// String formats a StatementFetchingResponse as a string
func (s *StatementFetchingResponse) String() string {
	if s == nil {
		return "StatementFetchingResponse=nil"
	}

	v, _ := s.Value()
	missingData := v.(MissingDataInStatement)
	return fmt.Sprintf("StatementFetchingResponse MissingDataInStatement=%+v", missingData)
}
