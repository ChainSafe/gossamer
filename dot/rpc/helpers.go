// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package rpc

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/rpc/v2"
)

// GetIP gets a requests IP address by reading off the forwarded-for
// header (for proxies) and falls back to use the remote address.
func GetIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

// LocalRequestOnly HTTP handler to restrict to only local connections
func LocalRequestOnly(r *rpc.RequestInfo, i interface{}) error {
	ip := GetIP(r.Request)
	if strings.Contains(ip, "127.0.0.1") || strings.Contains(ip, "localhost") {
		return nil
	}
	logger.Error("external HTTP request refuesed", "error")
	return errors.New("external HTTP request refused")
}
