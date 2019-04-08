package json2

import (
	"github.com/ChainSafe/gossamer/rpc"
	"log"
	"net"
	"net/http"
	"testing"
)

type NetService struct {
	version int
}

type NetArgs struct {}

type NetReply struct {
	value int
}

func(s *NetService) Version(r *http.Request, args *NetArgs, reply *NetReply) error {
	reply.value = s.version
	return nil
}


func StartRPCServer() {
	log.Println("Starting RPC server...")
	s := rpc.NewServer()
	log.Println("Started RPC server!")
	s.RegisterCodec(new(Codec))
	counter := new(NetService)
	err := s.RegisterService(counter, "")
	if err != nil {
		log.Fatalf("could not register service: %s", err)
	}
	log.Println("Registered service.")
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatalf("could not start listener: %s", err)
	}
	log.Println("Started listener.")
	err = http.Serve(l, s)
	if err != nil {
		log.Fatalf("Error serving: %s", err)
	}
	log.Println("Now serving requests :)")
}
//
//func MakeClientRequest() {
//	url := "http://localhost:3000/rpc"
//	args := &NetArgs{}
//	message, err := EncodeClientRequest("netservice_version", args)
//	if err != nil {
//		log.Fatalf("%s", err)
//	}
//	req, err := http.NewRequest("POST", url, bytes.NewBuffer(message))
//	if err != nil {
//		log.Fatalf("%s", err)
//	}
//	req.Header.Set("Content-Type", "application/json")
//	client := new(http.Client)
//	resp, err := client.Do(req)
//	if err != nil {
//		log.Fatalf("Error in sending request to %s. %s", url, err)
//	}
//	defer resp.Body.Close()
//
//	var result NetReply
//	err = json2.DecodeClientResponse(resp.Body, &result)
//	if err != nil {
//		log.Fatalf("Couldn't decode response. %s", err)
//	}
//	log.Printf("Response: %d", result.value)
//}

func TestServer(t *testing.T) {
	StartRPCServer()
	//MakeClientRequest()
}