package p2p

import (
	scale "github.com/ChainSafe/gossamer/codec"
	common "github.com/ChainSafe/gossamer/common"
	"testing"
)

func TestDecodeStatusMessage(t *testing.T) {
	genesisHash, err := common.HexToHash("0x9aa25e4c67a8a7e1d77572e4c3b97ca8110df952cfc3d345cec5e88cb1e3a96f")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash, err := common.HexToHash("0xb8d464ef3249ae1eea07db970e6e9bab74144dfa24e1a5a4132e4414a4a4c1ab")
	if err != nil {
		t.Fatal(err)
	}

	testStatus := &StatusMessage{
		ProtocolVersion: int32(112),
		Roles:           byte(2),
		BestBlockNumber: int64(2360405),
		BestBlockHash:   bestBlockHash,
		GenesisHash:     genesisHash,
		ChainStatus:     []byte{},
	}

	encStatus, err := scale.Encode(*testStatus)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(encStatus)

	encStatus, err = common.HexToBytes("0x00020000000200000004665624000000000064d2a1d642007a3f49b463ad46c335ea10ae6927b71496c132ea203311579761dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")
	if err != nil {
		t.Fatal(err)
	}

	rawMessage := RawMessage(append([]byte{0}, encStatus...))
	res, mType, err := rawMessage.Decode()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(res)
	t.Log(mType)
}
