package p2p

import (
	"bytes"
	"testing"

	"github.com/ChainSafe/gossamer/common"
)

func TestEncodeStatusMessage(t *testing.T) {
	genesisHash, err := common.HexToHash("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash, err := common.HexToHash("0x829de6be9a35b55c794c609c060698b549b3064c183504c18ab7517e41255569")
	if err != nil {
		t.Fatal(err)
	}

	testStatusMessage := &StatusMessage{
       ProtocolVersion: uint32(2),
       MinSupportedVersion: uint32(2),
       Roles:           byte(4),
       BestBlockNumber: uint64(2434417),
       BestBlockHash:   bestBlockHash,
       GenesisHash:     genesisHash,
       ChainStatus:     []byte{4, 0},
    }
    
  	buf := &bytes.Buffer{}
	l, err := testStatusMessage.Encode(buf)
	if err != nil {
		t.Fatal(err)
	}

	buf2 := make([]byte, l)
	_, err = buf.Read(buf2)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := common.HexToBytes("0x0200000002000000047125250000000000829de6be9a35b55c794c609c060698b549b3064c183504c18ab7517e41255569dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")

	if !bytes.Equal(buf2, expected) {
		t.Errorf("Fail: got %x expected %x", buf2, expected)
	}
}

func TestDecodeStatusMessage(t *testing.T) {
	if StatusMsg != 0 {
		t.Error("StatusMsg does not have correct underlying value")
	}

	genesisHash, err := common.HexToHash("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash, err := common.HexToHash("0x8dac4bd53582976cd2834b47d3c7b3a9c8c708db84b3bae145753547ec9ee4da")
	if err != nil {
		t.Fatal(err)
	}

	encStatus, err := common.HexToBytes("0x020000000200000004c1e72400000000008dac4bd53582976cd2834b47d3c7b3a9c8c708db84b3bae145753547ec9ee4dadcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	_, err = buf.Write(encStatus)
	if err != nil {
		t.Fatal(err)
	}

	sm := new(StatusMessage)
	err = sm.Decode(buf, uint64(len(encStatus)))
	if err != nil {
		t.Fatal(err)
	}

	if sm.ProtocolVersion != 2 {
		t.Error("did not get correct ProtocolVersion")
	} else if sm.MinSupportedVersion != 2 {
		t.Error("did not get correct MinSupportedVersion")
	} else if sm.Roles != byte(4) {
		t.Error("did not get correct Roles")
	} else if sm.BestBlockNumber != 2418625 {
		t.Error("did not get correct BestBlockNumber")
	} else if !bytes.Equal(sm.BestBlockHash.ToBytes(), bestBlockHash.ToBytes()) {
		t.Error("did not get correct BestBlockHash")
	} else if !bytes.Equal(sm.GenesisHash.ToBytes(), genesisHash.ToBytes()) {
		t.Error("did not get correct BestBlockHash")
	} else if !bytes.Equal(sm.ChainStatus, []byte{4, 0}) {
		t.Error("did not get correct ChainStatus")
	}

	t.Log(sm.String())
}
