package messages

import (
	"bytes"
	"fmt"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type WarpProofRequest struct {
	Begin common.Hash
}

func (wsr *WarpProofRequest) Decode(in []byte) error {
	reader := bytes.NewReader(in)
	sd := scale.NewDecoder(reader)
	reqProof := &WarpProofRequest{}
	err := sd.Decode(&reqProof)
	if err != nil {
		return err
	}

	return nil
}

func (wsr *WarpProofRequest) Encode() ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	encoder := scale.NewEncoder(buffer)
	err := encoder.Encode(wsr)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (wsr *WarpProofRequest) String() string {
	if wsr == nil {
		return "WarpProofRequest=nil"
	}

	return fmt.Sprintf("WarpProofRequest begin=%v", wsr.Begin)
}

var _ P2PMessage = (*WarpProofRequest)(nil)
