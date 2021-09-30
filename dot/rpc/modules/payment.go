package modules

import (
	"fmt"
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

type PaymentQueryInfoRequest struct {
	// hex SCALE encoded extrinsic
	Ext string
	// hex optional block hash indicating the state
	Hash common.Hash
}

type PaymentModule struct {
	blockAPI BlockAPI
}

func NewPaymentModule(blockAPI BlockAPI) *PaymentModule {
	return &PaymentModule{
		blockAPI: blockAPI,
	}
}

func (p *PaymentModule) QueryInfo(_ *http.Request, req *PaymentQueryInfoRequest, res *uint) error {
	if common.EmptyHash.Equal(req.Hash) {
		req.Hash = p.blockAPI.BestBlockHash()
	}

	r, err := p.blockAPI.GetRuntime(&req.Hash)
	if err != nil {
		return err
	}

	ext, err := common.HexToBytes(req.Ext)
	if err != nil {
		return err
	}

	encQueryInfo, err := r.PaymentQueryInfo(ext)
	if err != nil {
		return err
	}

	fmt.Println(encQueryInfo)

	return nil
}
