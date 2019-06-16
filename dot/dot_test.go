package dot

import (
	"github.com/ChainSafe/gossamer/common"
	cfg "github.com/ChainSafe/gossamer/config"
	api "github.com/ChainSafe/gossamer/internal"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
	"github.com/ChainSafe/gossamer/rpc/json2"
	"reflect"
	"testing"
)

var availableServices = [...]reflect.Type{
	reflect.TypeOf(p2p.Service{}),
	reflect.TypeOf(api.Service{}),
	reflect.TypeOf(polkadb.BadgerService{}),
}

// Creates a Dot with default configurations. Includes RPC server.
func createTestDot(t *testing.T) *Dot {
	var services []common.Service
	p2pSrvc, err := p2p.NewService(cfg.DefaultP2PConfig)
	services = append(services, p2pSrvc)
	if err != nil {
		t.Fatal(err)
	}

	// DB
	dataDir := "../test_data"
	dbSrvc, err := polkadb.NewBadgerService(dataDir)
	services = append(services, dbSrvc)
	if err != nil {
		t.Fatal(err)
	}

	// API
	apiSrvc := api.NewApiService(p2pSrvc)
	services = append(services, apiSrvc)
	// RPC
	rpcSrvc := rpc.NewHttpServer(apiSrvc, &json2.Codec{}, cfg.DefaultRpcConfig)

	return NewDot(services, rpcSrvc)
}

func TestDot_Start(t *testing.T) {
	dot := createTestDot(t)

	dot.Start()

	for _, srvc := range availableServices {
		s := dot.Services.Get(srvc)
		if s == nil {
			t.Fatalf("error getting service: %T", srvc)
		}
	}
}