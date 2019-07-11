package dot

import (
	"sync"
	"syscall"
	"testing"

	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
)

// Creates a Dot with default configurations. Does not include RPC server.
func createTestDot(t *testing.T) *Dot {
	var services []services.Service
	// P2P
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
	apiSrvc := api.NewApiService(p2pSrvc, nil)
	services = append(services, apiSrvc)

	return NewDot(services, nil)
}

func TestDot_Start(t *testing.T) {
	var availableServices = [...]services.Service{
		&p2p.Service{},
		&api.Service{},
		&polkadb.BadgerService{},
	}

	dot := createTestDot(t)

	go dot.Start(nil)

	// Wait until dot.Start() is finished
	<-dot.IsStarted

	for _, srvc := range availableServices {
		s := dot.Services.Get(srvc)
		if s == nil {
			t.Fatalf("error getting service: %T", srvc)
		}

		e := dot.Services.Err(srvc)
		if e == nil {
			t.Fatalf("error getting error channel for service: %T", srvc)
		}
	}

	dot.Stop()
}

func TestDot_StartIRQStop(t *testing.T) {
	var availableServices = [...]services.Service{
		&p2p.Service{},
		&api.Service{},
		&polkadb.BadgerService{},
	}

	dot := createTestDot(t)

	//WaitGroup to wait for all services to start before stopping them
	var wg sync.WaitGroup
	wg.Add(3)

	go dot.Start(&wg)
	wg.Wait()

	// Wait until dot.Start() is finished
	<-dot.IsStarted

	for _, srvc := range availableServices {
		s := dot.Services.Get(srvc)
		if s == nil {
			t.Fatalf("error getting service: %T", srvc)
		}

		e := dot.Services.Err(srvc)
		if e == nil {
			t.Fatalf("error getting error channel for service: %T", srvc)
		}
	}

	//Send a interrupt signal to kill all services
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	//Wait for stop IRQ signal & finish test
	<-dot.stop
	dot.Stop()
}
