package parachain

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/collation_protocol.yaml
var expectedCollationProtocolHexRaw string

var expectedCollationProtocolHex map[string]string

func init() {
	err := yaml.Unmarshal([]byte(expectedCollationProtocolHexRaw), &expectedCollationProtocolHex)
	if err != nil {
		fmt.Println("Error unmarshaling test data:", err)
		return
	}
}

func TestCollationProtocol(t *testing.T) {
	t.Parallel()

	var collatorID CollatorID
	tempCollatID := common.MustHexToBytes("0x48215b9d322601e5b1a95164cea0dc4626f545f98343d07f1551eb9543c4b147")
	copy(collatorID[:], tempCollatID)

	var collatorSignature CollatorSignature
	tempSignature := common.MustHexToBytes("0xc67cb93bf0a36fcee3d29de8a6a69a759659680acf486475e0a2552a5fbed87e45adce5f290698d8596095722b33599227f7461f51af8617c8be74b894cf1b86") //nolint:lll
	copy(collatorSignature[:], tempSignature)

	var validatorSignature ValidatorSignature
	copy(validatorSignature[:], tempSignature)

	hash5 := getDummyHash(5)

	secondedEnumValue := Seconded{
		Descriptor: CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 hash5,
			Collator:                    collatorID,
			PersistedValidationDataHash: hash5,
			PovHash:                     hash5,
			ErasureRoot:                 hash5,
			Signature:                   collatorSignature,
			ParaHead:                    hash5,
			ValidationCodeHash:          ValidationCodeHash(hash5),
		},
		Commitments: CandidateCommitments{
			UpwardMessages:            []UpwardMessage{{1, 2, 3}},
			HorizontalMessages:        []OutboundHrmpMessage{},
			NewValidationCode:         &ValidationCode{1, 2, 3},
			HeadData:                  headData{1, 2, 3},
			ProcessedDownwardMessages: uint32(5),
			HrmpWatermark:             uint32(0),
		},
	}

	statementWithSeconded := NewStatement()
	err := statementWithSeconded.Set(secondedEnumValue)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		enumValue     scale.VaryingDataTypeValue
		encodingValue []byte // encoding of CollationProtocol value
	}{
		{
			name: "Declare",
			enumValue: Declare{
				CollatorId:        collatorID,
				ParaId:            uint32(5),
				CollatorSignature: collatorSignature,
			},
			encodingValue: common.MustHexToBytes(expectedCollationProtocolHex["declare"]),
		},
		{
			name:          "AdvertiseCollation",
			enumValue:     AdvertiseCollation(hash5),
			encodingValue: common.MustHexToBytes("0x00010505050505050505050505050505050505050505050505050505050505050505"),
		},
		{
			name: "CollationSeconded",
			enumValue: CollationSeconded{
				Hash: hash5,
				UncheckedSignedFullStatement: UncheckedSignedFullStatement{
					Payload:        statementWithSeconded,
					ValidatorIndex: ValidatorIndex(5),
					Signature:      validatorSignature,
				},
			},
			encodingValue: common.MustHexToBytes(expectedCollationProtocolHex["collationSeconded"]),
		},
	}

	for _, c := range testCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			vdt_parent := NewCollationProtocol()
			vdt_child := NewCollatorProtocolMessage()

			err := vdt_child.Set(c.enumValue)
			require.NoError(t, err)

			err = vdt_parent.Set(vdt_child)
			require.NoError(t, err)

			bytes, err := scale.Marshal(vdt_parent)
			require.NoError(t, err)

			require.Equal(t, c.encodingValue, bytes)
		})
	}
}
