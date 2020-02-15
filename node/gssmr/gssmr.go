// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package gssmr

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ChainSafe/gossamer/internal/services"
	"github.com/ChainSafe/gossamer/rpc"
	log "github.com/ChainSafe/log15"
)

// Node is a container for all the components of a node.
type Node struct {
	Name      string
	Services  *services.ServiceRegistry // Registry of all core services
	RPC       *rpc.HTTPServer           // HTTP instance for RPC server
	IsStarted chan struct{}             // Signals node startup complete
	stop      chan struct{}             // Used to signal node shutdown
}

// NewNode initializes a Node with provided components.
func NewNode(name string, srvcs []services.Service, rpc *rpc.HTTPServer) *Node {
	d := &Node{
		Name:      name,
		Services:  services.NewServiceRegistry(),
		RPC:       rpc,
		IsStarted: make(chan struct{}),
		stop:      nil,
	}

	for _, srvc := range srvcs {
		d.Services.RegisterService(srvc)
	}

	return d
}

// Start starts all services. API service is started last.
func (d *Node) Start() {
	d.Services.StartAll()
	if d.RPC != nil {
		d.RPC.Start()
	}

	d.stop = make(chan struct{})
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		d.Stop()
		os.Exit(130)
	}()

	//Move on when routine catches SIGINT or SIGTERM calls
	close(d.IsStarted)
	d.Wait()
}

// Wait is used to force the node to stay alive until a signal is passed into `Node.stop`
func (d *Node) Wait() {
	<-d.stop
}

//Stop all services first, then send stop signal for test
func (d *Node) Stop() {
	d.Services.StopAll()
	if d.stop != nil {
		close(d.stop)
	}
}
