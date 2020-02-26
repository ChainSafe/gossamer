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

package network

import (
	"os"
	"path"
	"reflect"
	"testing"
	"time"
)

// test host connect method
func TestConnect(t *testing.T) {
	dataDirA := path.Join(os.TempDir(), "gossamer-test", "nodeA")
	defer os.RemoveAll(dataDirA)

	configA := &Config{
		DataDir:     dataDirA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeA, _, _ := createTestService(t, configA)
	defer nodeA.Stop()

	nodeA.noGossip = true
	nodeA.noStatus = true

	dataDirB := path.Join(os.TempDir(), "gossamer-test", "nodeB")
	defer os.RemoveAll(dataDirB)

	configB := &Config{
		DataDir:     dataDirB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeB, _, _ := createTestService(t, configB)
	defer nodeB.Stop()

	nodeB.noGossip = true
	nodeB.noStatus = true

	addrInfosB, err := nodeB.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosB[0])
	if err != nil {
		t.Fatal(err)
	}

	peerCountA := nodeA.host.peerCount()
	peerCountB := nodeB.host.peerCount()

	if peerCountA != 1 {
		t.Error(
			"node A does not have expected peer count",
			"\nexpected:", 1,
			"\nreceived:", peerCountA,
		)
	}

	if peerCountB != 1 {
		t.Error(
			"node B does not have expected peer count",
			"\nexpected:", 1,
			"\nreceived:", peerCountB,
		)
	}
}

// test host bootstrap method on start
func TestBootstrap(t *testing.T) {
	dataDirA := path.Join(os.TempDir(), "gossamer-test", "nodeA")
	defer os.RemoveAll(dataDirA)

	configA := &Config{
		DataDir:     dataDirA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeA, _, _ := createTestService(t, configA)
	defer nodeA.Stop()

	nodeA.noGossip = true
	nodeA.noStatus = true

	addrA := nodeA.host.multiaddrs()[0]

	dataDirB := path.Join(os.TempDir(), "gossamer-test", "nodeB")
	defer os.RemoveAll(dataDirB)

	configB := &Config{
		DataDir:   dataDirB,
		Port:      7002,
		RandSeed:  2,
		Bootnodes: []string{addrA.String()},
		NoMdns:    true,
	}

	nodeB, _, _ := createTestService(t, configB)
	defer nodeB.Stop()

	nodeB.noGossip = true
	nodeB.noStatus = true

	peerCountA := nodeA.host.peerCount()
	peerCountB := nodeB.host.peerCount()

	if peerCountA != 1 {
		t.Error(
			"node A does not have expected peer count",
			"\nexpected:", 1,
			"\nreceived:", peerCountA,
		)
	}

	if peerCountB != 1 {
		t.Error(
			"node B does not have expected peer count",
			"\nexpected:", 1,
			"\nreceived:", peerCountB,
		)
	}
}

// test host ping method
func TestPing(t *testing.T) {
	dataDirA := path.Join(os.TempDir(), "gossamer-test", "nodeA")
	defer os.RemoveAll(dataDirA)

	configA := &Config{
		DataDir:     dataDirA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeA, _, _ := createTestService(t, configA)
	defer nodeA.Stop()

	nodeA.noGossip = true
	nodeA.noStatus = true

	dataDirB := path.Join(os.TempDir(), "gossamer-test", "nodeB")
	defer os.RemoveAll(dataDirB)

	configB := &Config{
		DataDir:     dataDirB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeB, _, _ := createTestService(t, configB)
	defer nodeB.Stop()

	nodeB.noGossip = true
	nodeB.noStatus = true

	addrInfosB, err := nodeB.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosB[0])
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.ping(addrInfosB[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	addrInfosA, err := nodeA.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeB.host.ping(addrInfosA[0].ID)
	if err != nil {
		t.Fatal(err)
	}
}

// test host send method
func TestSend(t *testing.T) {
	dataDirA := path.Join(os.TempDir(), "gossamer-test", "nodeA")
	defer os.RemoveAll(dataDirA)

	configA := &Config{
		DataDir:     dataDirA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeA, _, _ := createTestService(t, configA)
	defer nodeA.Stop()

	nodeA.noGossip = true
	nodeA.noStatus = true

	dataDirB := path.Join(os.TempDir(), "gossamer-test", "nodeB")
	defer os.RemoveAll(dataDirB)

	configB := &Config{
		DataDir:     dataDirB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeB, msgSendB, _ := createTestService(t, configB)
	defer nodeB.Stop()

	nodeB.noGossip = true
	nodeB.noStatus = true

	addrInfosB, err := nodeB.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosB[0])
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.send(addrInfosB[0].ID, TestMessage)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-msgSendB:
		if !reflect.DeepEqual(msg, TestMessage) {
			t.Error(
				"node B received unexpected message from node A",
				"\nexpected:", TestMessage,
				"\nreceived:", msg,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("node B timeout waiting for message from node A")
	}
}

// test host broadcast method
func TestBroadcast(t *testing.T) {
	dataDirA := path.Join(os.TempDir(), "gossamer-test", "nodeA")
	defer os.RemoveAll(dataDirA)

	configA := &Config{
		DataDir:     dataDirA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeA, _, _ := createTestService(t, configA)
	defer nodeA.Stop()

	nodeA.noGossip = true
	nodeA.noStatus = true

	dataDirB := path.Join(os.TempDir(), "gossamer-test", "nodeB")
	defer os.RemoveAll(dataDirB)

	configB := &Config{
		DataDir:     dataDirB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeB, msgSendB, _ := createTestService(t, configB)
	defer nodeB.Stop()

	nodeB.noGossip = true
	nodeB.noStatus = true

	addrInfosB, err := nodeB.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosB[0])
	if err != nil {
		t.Fatal(err)
	}

	dataDirC := path.Join(os.TempDir(), "gossamer-test", "nodeC")
	defer os.RemoveAll(dataDirC)

	configC := &Config{
		DataDir:     dataDirC,
		Port:        7003,
		RandSeed:    3,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeC, msgSendC, _ := createTestService(t, configC)
	defer nodeC.Stop()

	nodeC.noGossip = true
	nodeC.noStatus = true

	addrInfosC, err := nodeC.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosC[0])
	if err != nil {
		t.Fatal(err)
	}

	nodeA.host.broadcast(TestMessage)

	select {
	case msg := <-msgSendB:
		if !reflect.DeepEqual(msg, TestMessage) {
			t.Error(
				"node B received unexpected message from node A",
				"\nexpected:", TestMessage,
				"\nreceived:", msg,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("node B timeout waiting for message")
	}

	select {
	case msg := <-msgSendC:
		if !reflect.DeepEqual(msg, TestMessage) {
			t.Error(
				"node C received unexpected message from node A",
				"\nexpected:", TestMessage,
				"\nreceived:", msg,
			)
		}
	case <-time.After(TestMessageTimeout):
		t.Error("node C timeout waiting for message")
	}

}

// test host send method with existing stream
func TestExistingStream(t *testing.T) {
	dataDirA := path.Join(os.TempDir(), "gossamer-test", "nodeA")
	defer os.RemoveAll(dataDirA)

	configA := &Config{
		DataDir:     dataDirA,
		Port:        7001,
		RandSeed:    1,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeA, msgSendA, _ := createTestService(t, configA)
	defer nodeA.Stop()

	nodeA.noGossip = true
	nodeA.noStatus = true

	addrInfosA, err := nodeA.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	dataDirB := path.Join(os.TempDir(), "gossamer-test", "nodeB")
	defer os.RemoveAll(dataDirB)

	configB := &Config{
		DataDir:     dataDirB,
		Port:        7002,
		RandSeed:    2,
		NoBootstrap: true,
		NoMdns:      true,
	}

	nodeB, msgSendB, _ := createTestService(t, configB)
	defer nodeB.Stop()

	nodeB.noGossip = true
	nodeB.noStatus = true

	addrInfosB, err := nodeB.host.addrInfos()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeA.host.connect(*addrInfosB[0])
	if err != nil {
		t.Fatal(err)
	}

	stream := nodeA.host.getStream(nodeB.host.id())
	if stream != nil {
		t.Error("node A should not have an outbound stream")
	}

	// node A opens the stream to send the first message
	err = nodeA.host.send(addrInfosB[0].ID, TestMessage)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-msgSendB:
	case <-time.After(TestMessageTimeout):
		t.Error("node B timeout waiting for message from node A")
	}

	stream = nodeA.host.getStream(nodeB.host.id())
	if stream == nil {
		t.Error("node A should have an outbound stream")
	}

	// node A uses the stream to send a second message
	err = nodeA.host.send(addrInfosB[0].ID, TestMessage)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-msgSendB:
	case <-time.After(TestMessageTimeout):
		t.Error("node B timeout waiting for message from node A")
	}

	stream = nodeA.host.getStream(nodeB.host.id())
	if stream == nil {
		t.Error("node A should have an outbound stream")
	}

	stream = nodeB.host.getStream(nodeA.host.id())
	if stream != nil {
		t.Error("node B should not have an outbound stream")
	}

	// node B opens the stream to send the first message
	err = nodeB.host.send(addrInfosA[0].ID, TestMessage)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-msgSendA:
	case <-time.After(TestMessageTimeout):
		t.Error("node A timeout waiting for message from node B")
	}

	stream = nodeB.host.getStream(nodeA.host.id())
	if stream == nil {
		t.Error("node B should have an outbound stream")
	}

	// node B uses the stream to send a second message
	err = nodeB.host.send(addrInfosA[0].ID, TestMessage)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-msgSendA:
	case <-time.After(TestMessageTimeout):
		t.Error("node A timeout waiting for message from node B")
	}

	stream = nodeB.host.getStream(nodeA.host.id())
	if stream == nil {
		t.Error("node B should have an outbound stream")
	}
}
