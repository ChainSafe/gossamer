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
	"fmt"
	"net"
	"strings"

	"github.com/ChainSafe/gossamer/dot/rpc/modules"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/rpc/v2"
	"github.com/jpillora/ipfilter"
)

// LocalhostFilter creates a ipfilter object for localhost
func LocalhostFilter() *ipfilter.IPFilter {
	return ipfilter.New(ipfilter.Options{
		BlockByDefault: true,
		AllowedIPs:     []string{"127.0.0.1", "::1"},
	})
}

// LocalRequestOnly HTTP handler to restrict to only local connections
func LocalRequestOnly(r *rpc.RequestInfo, i interface{}) error {
	ip, _, err := net.SplitHostPort(r.Request.RemoteAddr)

	if err != nil {
		return errors.New("unable to parse IP")
	}
	f := LocalhostFilter()
	if allowed := f.Allowed(ip); allowed {
		return nil
	}
	return errors.New("external HTTP request refused")
}

func snakeCaseFormat(method string) (string, error) {
	parts := strings.Split(method, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid rpc method format %s, should be 'module.FunctionName'", method)
	}

	service, funcName := parts[0], parts[1]
	funcName = strings.ToLower(string(funcName[0])) + funcName[1:]
	return strings.Join([]string{service, funcName}, "_"), nil
}

func rpcValidator(cfg *HTTPServerConfig, validate *validator.Validate) func(r *rpc.RequestInfo, i interface{}) error {
	return func(r *rpc.RequestInfo, v interface{}) error {
		var (
			err       error
			rpcmethod string
		)

		if rpcmethod, err = snakeCaseFormat(r.Method); err != nil {
			return err
		}

		isUnsafe := modules.IsUnsafe(rpcmethod)
		if isUnsafe && !cfg.rpcUnsafeEnabled() {
			return fmt.Errorf("unsafe rpc method %s cannot be reachable", rpcmethod)
		}

		if err = validate.Struct(v); err != nil {
			return err
		}

		if !cfg.exposeRPC() || modules.IsUnsafe(rpcmethod) && !cfg.RPCUnsafeExternal {
			return LocalRequestOnly(r, v)
		}

		return nil
	}
}
