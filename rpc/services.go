package rpc

import (
	"github.com/ChainSafe/gossamer/p2p"
	"log"
	"net/http"
)

type ArgsP2P struct {}

type ReplyP2P struct {
	Count int
}

type P2PService struct {
	Srv *p2p.Service
}

func (p *P2PService) PeerCount(r *http.Request, args *ArgsP2P, reply *ReplyP2P) error {
	reply.Count = p.Srv.PeerCount()
	log.Printf("PeerCount -- Got N: %d", reply.Count)
	return nil
}
