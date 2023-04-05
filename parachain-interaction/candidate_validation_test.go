package parachaininteraction

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

func TestValidateFromChainState(t *testing.T) {
	// CandidateReceipt { descriptor: CandidateDescriptor { para_id: Id(1), relay_parent: 0x0505050505050505050505050505050505050505050505050505050505050505, collator: Public(0000000000000000000000000000000000000000000000000000000000000000 (5C4hrfjw...)), persisted_validation_data_hash: 0x0000000000000000000000000000000000000000000000000000000000000000, pov_hash: 0xfa924fcc5dc5a9177aa1b75447e00b764ecb758ca55a49d45eb6d08f2cf1dc56, erasure_root: 0x1d779e257123dbf107da481d4a08528cd469371f18b7d10f789e475f61206279, signature: Signature(00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000), para_head: 0x0000000000000000000000000000000000000000000000000000000000000000, validation_code_hash: 0x11c0e79b71c3976ccd0c02d1310e2516c08edc9d8b6f57ccd680d63a4d8e72da }, commitments_hash: 0x1c60d15e03474774fb41adf24a7f7185d72ca7401c79a35b0dcfc3b168565b78 }

	collatorID, _ := sr25519.NewPublicKey([]byte("5C4hrfjw9DjXZTzV3MwzrrAr9P1MJhSrvWGWqi1eSuyUpnhM"))
	candidateReceipt := CandidateReceipt{
		descriptor: CandidateDescriptor{
			ParaID:                      uint32(1),
			RelayParent:                 common.MustHexToHash("0x0505050505050505050505050505050505050505050505050505050505050505"),
			Collator:                    *collatorID,
			PersistedValidationDataHash: common.MustHexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			PovHash:                     common.MustHexToHash("0xfa924fcc5dc5a9177aa1b75447e00b764ecb758ca55a49d45eb6d08f2cf1dc56"),
			ErasureRoot:                 common.MustHexToHash("0x1d779e257123dbf107da481d4a08528cd469371f18b7d10f789e475f61206279"),
			Signature:                   collatorSignature{},
			ParaHead:                    common.MustHexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
			ValidationCodeHash:          validationCodeHash(common.MustHexToHash("0x11c0e79b71c3976ccd0c02d1310e2516c08edc9d8b6f57ccd680d63a4d8e72da")),
		},
		commitmentsHash: common.MustHexToHash("0x1c60d15e03474774fb41adf24a7f7185d72ca7401c79a35b0dcfc3b168565b78"),
	}
}
