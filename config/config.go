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

package cfg

import (
	"github.com/ChainSafe/gossamer/p2p"
	"github.com/ChainSafe/gossamer/polkadb"
	"github.com/ChainSafe/gossamer/rpc"
)

// Config is a collection of configurations throughout the system
type Config struct {
	P2PConfig 	*p2p.Config
	DbConfig    *polkadb.Config
	RPCConfig	*rpc.Config
}


// CheckConfig finds file based on ext input
// TODO: Remove?
//func CheckConfig(ext string) string {
//	pathS, err := os.Getwd()
//	if err != nil {
//		panic(err)
//	}
//	var file string
//	if err = filepath.Walk(pathS, func(path string, f os.FileInfo, _ error) error {
//		if !f.IsDir() && f.Name() == "config.toml" {
//			r, e := regexp.MatchString(ext, f.Name())
//			if e == nil && r {
//				file = f.Name()
//				return nil
//			}
//		} else if !f.IsDir() && f.Name() != "Gopkg.toml" {
//			r, e := regexp.MatchString(ext, f.Name())
//			if e == nil && r {
//				file = f.Name()
//				return nil
//			}
//		}
//		return nil
//	}); err != nil {
//		log.Error("please specify a config file", "err", err)
//	}
//	return file
//}