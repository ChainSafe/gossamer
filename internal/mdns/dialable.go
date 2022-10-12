// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package mdns

import (
	"errors"
	"fmt"
	"net"

	manet "github.com/multiformats/go-multiaddr/net"
)

var (
	ErrTCPListenAddressNotFound = errors.New("TCP listen address not found")
)

func getMDNSIPsAndPort(p2pHost Networker) (ips []net.IP, port uint16, err error) {
	tcpAddresses, err := getDialableListenAddrs(p2pHost)
	if err != nil {
		return nil, 0, fmt.Errorf("getting dialable listen addresses: %w", err)
	}

	ips = make([]net.IP, len(tcpAddresses))
	for i := range tcpAddresses {
		ips[i] = tcpAddresses[i].IP
	}
	port = uint16(tcpAddresses[0].Port)

	return ips, port, nil
}

func getDialableListenAddrs(p2pHost Networker) (tcpAddresses []*net.TCPAddr, err error) {
	multiAddresses, err := p2pHost.Network().InterfaceListenAddresses()
	if err != nil {
		return nil, fmt.Errorf("listing host interface listen addresses: %w", err)
	}

	tcpAddresses = make([]*net.TCPAddr, 0, len(multiAddresses))
	for _, multiAddress := range multiAddresses {
		netAddress, err := manet.ToNetAddr(multiAddress)
		if err != nil {
			continue
		}

		tcpAddress, ok := netAddress.(*net.TCPAddr)
		if !ok {
			continue
		}

		tcpAddresses = append(tcpAddresses, tcpAddress)
	}

	if len(tcpAddresses) == 0 {
		return nil, fmt.Errorf("%w: in %d multiaddresses", ErrTCPListenAddressNotFound, len(multiAddresses))
	}

	return tcpAddresses, nil
}
