package p2p

import (
	"bytes"
	//"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/common"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

func TestAlexander(t *testing.T) {
	testServiceConfig := &Config{
		BootstrapNodes: []string{
			"/ip4/104.211.54.233/tcp/30363/p2p/16Uiu2HAmFWPUx45xYYeCpAryQbvU3dY8PWGdMwS2tLm1dB1CsmCj",
			"/ip4/104.211.48.51/tcp/30363/p2p/16Uiu2HAmJqVCtF5oMvu1rbJvqWubMMRuWiKJtpoM8KSQ3JNnL5Ec",
			"/ip4/104.211.48.247/tcp/30363/p2p/16Uiu2HAkyhNWHTPcA2dVKzMnLpFebXqsDQMpkuGnS9SqjJyDyULi",
			"/ip4/40.117.153.33/tcp/30363/p2p/16Uiu2HAmKXzRnzgyVtSyyp6ozAk5aT9H7PEi2ozkHSzzg7vmX7LV",
		},
		Port: 30304,
	}

	sb, err := NewService(testServiceConfig)
	if err != nil {
		t.Fatalf("NewService error: %s", err)
	}

	// go func(s *Service) {
	//     for {
	//         fmt.Printf("PeerStore size %d\n",len(s.Host().Peerstore().Peers()))
	//         fmt.Printf("PeerCount %d\n", s.PeerCount())
	//         time.Sleep(time.Second * 5)
	//     }
	// }(sb)

	e := sb.Start()
	err = <-e
	if err != nil {
		t.Errorf("Start error: %s", err)
	}

	time.Sleep(5 * time.Second)

	pid, err := peer.IDB58Decode("16Uiu2HAkyhNWHTPcA2dVKzMnLpFebXqsDQMpkuGnS9SqjJyDyULi")
	if err != nil {
		t.Fatal(err)
	}

	p, err := sb.dht.FindPeer(sb.ctx, pid)
	if err != nil {
		t.Fatalf("could not find peer: %s", err)
	}

	status, err := common.HexToBytes("0x000200000002000000049fbc25000000000066bece5466eec2d5d6ad53a296cc9a2469cf063970971c0049c07c891cfb701cdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")
	if err != nil {
		t.Fatal(err)
	}

	err = sb.Send(p, status)
	if err != nil {
		t.Error(err)
	}

	// time.Sleep(2 * time.Second)

	// genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// endBlock, err := common.HexToHash("0x9aa25e4c67a8a7e1d77572e4c3b97ca8110df952cfc3d345cec5e88cb1e3a96f")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	bm := &BlockRequestMessage{
		Id:            15,
		RequestedData: 1,
		StartingBlock: []byte{1, 1},// append([]byte{0}, genesisHash...),
		//StartingBlock: genesisHash,
		//EndBlockHash:  endBlock,
		Direction:     1,
		//Max:           2,
	}

	msg, err := bm.Encode()
	if err != nil {
		t.Fatal(err)
	}

	// stream := sb.GetExistingStream(pid)
	// if stream != nil {
	// 	fmt.Printf("using existing stream to send block request...\n")
	// 	_, err = stream.Write(msg)
	// 	if err != nil {
	// 		fmt.Printf("write to stream err %s", err)
	// 	}
	// } else {
	// 	fmt.Printf("using new stream to send block request...\n")
	// 	stream, err = sb.host.NewStream(sb.ctx, pid, protocolPrefix2)
	// 	if err != nil {
	// 		fmt.Printf("new stream err %s", err)
	// 	}
	// 	_, err = stream.Write(msg)
	// 	if err != nil {
	// 		fmt.Printf("write to stream err %s", err)
	// 	}	
	// }

	err = sb.Send(p, msg)
	if err != nil {
		t.Errorf("Send error: %s", err)
	}

	// pid, err = peer.IDB58Decode("16Uiu2HAkyhNWHTPcA2dVKzMnLpFebXqsDQMpkuGnS9SqjJyDyULi")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// p, err = sb.dht.FindPeer(sb.ctx, pid)
	// if err != nil {
	// 	t.Fatalf("could not find peer: %s", err)
	// }

	// err = sb.Send(p, msg)
	// if err != nil {
	// 	t.Errorf("Send error: %s", err)
	// }

	select {}
}

func TestDecodeMessageStatus(t *testing.T) {
	encMsg, err := common.HexToBytes("0x00020000000200000004c1e72400000000008dac4bd53582976cd2834b47d3c7b3a9c8c708db84b3bae145753547ec9ee4dadcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	buf.Write(encMsg)

	m, err := DecodeMessage(buf)
	if err != nil {
		t.Fatal(err)
	}

	genesisHash, err := common.HexToHash("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash, err := common.HexToHash("0x8dac4bd53582976cd2834b47d3c7b3a9c8c708db84b3bae145753547ec9ee4da")
	if err != nil {
		t.Fatal(err)
	}

	sm := m.(*StatusMessage)
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
	} else if !bytes.Equal(sm.ChainStatus, []byte{0}) {
		t.Error("did not get correct ChainStatus")
	}
}

func TestDecodeMessageBlockRequest(t *testing.T) {
	encMsg, err := common.HexToBytes("0x01611e0000018400dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025bfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa101ffffffff")
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	buf.Write(encMsg)

	m, err := DecodeMessage(buf)
	if err != nil {
		t.Fatal(err)
	}

	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &BlockRequestMessage{
		Id:            7777,
		RequestedData: 1,
		StartingBlock: append([]byte{0}, genesisHash...),
		EndBlockHash:  endBlock,
		Direction:     1,
		Max:           1<<32 - 1,
	}

	bm := m.(*BlockRequestMessage)
	if !reflect.DeepEqual(bm, expected) {
		t.Fatalf("Fail: got %v expected %v", bm, expected)
	}
}

func TestDecodeStatusMessage(t *testing.T) {
	encStatus, err := common.HexToBytes("0x020000000200000004c1e72400000000008dac4bd53582976cd2834b47d3c7b3a9c8c708db84b3bae145753547ec9ee4dadcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")
	if err != nil {
		t.Fatal(err)
	}

	genesisHash, err := common.HexToHash("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash, err := common.HexToHash("0x8dac4bd53582976cd2834b47d3c7b3a9c8c708db84b3bae145753547ec9ee4da")
	if err != nil {
		t.Fatal(err)
	}

	sm := new(StatusMessage)
	err = sm.Decode(encStatus)
	if err != nil {
		t.Fatal(err)
	}

	expected := &StatusMessage{
		ProtocolVersion:     uint32(2),
		MinSupportedVersion: uint32(2),
		Roles:               byte(4),
		BestBlockNumber:     uint64(2418625),
		BestBlockHash:       bestBlockHash,
		GenesisHash:         genesisHash,
		ChainStatus:         []byte{0},
	}

	if !reflect.DeepEqual(sm, expected) {
		t.Fatalf("Fail: got %v expected %v", sm, expected)
	}
}

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
		ProtocolVersion:     uint32(2),
		MinSupportedVersion: uint32(2),
		Roles:               byte(4),
		BestBlockNumber:     uint64(2434417),
		BestBlockHash:       bestBlockHash,
		GenesisHash:         genesisHash,
		ChainStatus:         []byte{0},
	}

	encStatus, err := testStatusMessage.Encode()
	if err != nil {
		t.Fatal(err)
	}

	expected, err := common.HexToBytes("0x000200000002000000047125250000000000829de6be9a35b55c794c609c060698b549b3064c183504c18ab7517e41255569dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b0400")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(encStatus, expected) {
		t.Errorf("Fail: got %x expected %x", encStatus, expected)
	}
}

func TestDecodeBlockRequestMessage(t *testing.T) {
	encMsg, err := common.HexToBytes("0x611e0000018400dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025bfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa101ffffffff")
	if err != nil {
		t.Fatal(err)
	}

	bm := new(BlockRequestMessage)
	err = bm.Decode(encMsg)
	if err != nil {
		t.Fatal(err)
	}

	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	if err != nil {
		t.Fatal(err)
	}

	expected := &BlockRequestMessage{
		Id:            7777,
		RequestedData: 1,
		StartingBlock: append([]byte{0}, genesisHash...),
		EndBlockHash:  endBlock,
		Direction:     1,
		Max:           1<<32 - 1,
	}

	if !reflect.DeepEqual(bm, expected) {
		t.Fatalf("Fail: got %v expected %v", bm, expected)
	}
}

func TestEncodeBlockRequestMessage(t *testing.T) {
	expected, err := common.HexToBytes("0x01611e0000018400dcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025bfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa101ffffffff")
	if err != nil {
		t.Fatal(err)
	}

	genesisHash, err := common.HexToBytes("0xdcd1346701ca8396496e52aa2785b1748deb6db09551b72159dcb3e08991025b")
	if err != nil {
		t.Fatal(err)
	}

	endBlock, err := common.HexToHash("0xfd19d9ebac759c993fd2e05a1cff9e757d8741c2704c8682c15b5503496b6aa1")
	if err != nil {
		t.Fatal(err)
	}

	bm := &BlockRequestMessage{
		Id:            7777,
		RequestedData: 1,
		StartingBlock: append([]byte{0}, genesisHash...),
		EndBlockHash:  endBlock,
		Direction:     1,
		Max:           1<<32 - 1,
	}

	encMsg, err := bm.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(encMsg, expected) {
		t.Fatalf("Fail: got %x expected %x", encMsg, expected)
	}
}
