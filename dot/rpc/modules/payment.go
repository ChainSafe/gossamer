package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

type PaymentQueryInfoRequest struct {
	// hex SCALE encoded extrinsic
	Ext string
	// hex optional block hash indicating the state
	Hash *common.Hash
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
	var hash common.Hash
	if req.Hash == nil {
		hash = p.blockAPI.BestBlockHash()
	} else {
		hash = *req.Hash
	}

	r, err := p.blockAPI.GetRuntime(&hash)
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

	if encQueryInfo != nil {
		*res = encQueryInfo.PartialFee
	}

	return nil
}
