package extrinsic

import (
	"testing"
)

func TestTransferExt_Encode(t *testing.T) {
	transfer := NewTransfer(8, 9, 1000, 1)
	sig := [64]byte{}
	ext := NewTransferExt(transfer, sig, false)

	enc, err := ext.Encode()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(ext)
	t.Log(enc)
}
