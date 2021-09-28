package modules

import (
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
)

type PaymentQueryInfo struct {
	// hex SCALE encoded extrinsic
	Ext string
	// hex optional block hash indicating the state
	Hash common.Hash
}

type Payment struct {
}

func NewPaymentModule() *Payment {
	return &Payment{}
}

func (p *Payment) QueryInfo(_ *http.Request, req *PaymentQueryInfo, res *uint) error {
	return nil
}
