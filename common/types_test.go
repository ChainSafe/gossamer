package common

import (
	"math/big"
	"testing"
)

// func TestHashEmptyBlockHeader(t *testing.T) {
// 	header := &BlockHeader{
// 		ParentHash: [32]byte{},
// 		Number: big.NewInt(0),
// 		StateRoot: [32]byte{},
// 		ExtrinsicsRoot: [32]byte{},
// 		//Digest: []byte{},
// 	}

// 	hash, err := header.Hash()
// 	if err != nil {
// 		t.Errorf("Fail: could not hash block header: %s", err)
// 	}

// 	t.Log(hash)
// }

// func TestHashBlockHeader(t *testing.T) {
// 	ph, err := HexToHash("0x8550326cee1e1b768a254095b412e0db58523c2b5df9b7d2540b4513d475ce7f")
// 	if err != nil {
// 		t.Fatalf("Fail when decoding parent hash: %s", err)
// 	}
// 	sr, err := HexToHash("0x1d9d01423a90032ac600d1e2ff0a54634760d0ae0941cfab855c69bef38689d2")
// 	if err != nil {
// 		t.Fatalf("Fail when decoding state root: %s", err)
// 	}	
// 	er, err := HexToHash("0x118a02e06882254b1d24417d4df4dca6a7b8754e42f5b24419f7170a0de6d027")
// 	if err != nil {
// 		t.Fatalf("Fail when decoding extrinsics root: %s", err)
// 	}

// 	header := &BlockHeader{
// 		ParentHash: ph,
// 		Number: big.NewInt(1570578),
// 		StateRoot: sr,
// 		ExtrinsicsRoot: er,
// 		Digest: []byte{},
// 	}

// 	hash, err := header.Hash()
// 	if err != nil {
// 		t.Errorf("Fail: could not hash block header: %s", err)
// 	}
	
// 	t.Logf("%x", hash)
// }