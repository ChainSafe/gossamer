// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only


package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path"
	"reflect"

	"github.com/golang/mock/mockgen/model"

	pkg_ "github.com/ChainSafe/gossamer/dot/rpc/modules"
)

var output = flag.String("output", "", "The output file name, or empty to use stdout.")

func main() {
	flag.Parse()

	its := []struct{
		sym string
		typ reflect.Type
	}{
		
		{ "StorageAPI", reflect.TypeOf((*pkg_.StorageAPI)(nil)).Elem()},
		
		{ "BlockAPI", reflect.TypeOf((*pkg_.BlockAPI)(nil)).Elem()},
		
		{ "NetworkAPI", reflect.TypeOf((*pkg_.NetworkAPI)(nil)).Elem()},
		
		{ "BlockProducerAPI", reflect.TypeOf((*pkg_.BlockProducerAPI)(nil)).Elem()},
		
		{ "TransactionStateAPI", reflect.TypeOf((*pkg_.TransactionStateAPI)(nil)).Elem()},
		
		{ "CoreAPI", reflect.TypeOf((*pkg_.CoreAPI)(nil)).Elem()},
		
		{ "SystemAPI", reflect.TypeOf((*pkg_.SystemAPI)(nil)).Elem()},
		
		{ "BlockFinalityAPI", reflect.TypeOf((*pkg_.BlockFinalityAPI)(nil)).Elem()},
		
		{ "RuntimeStorageAPI", reflect.TypeOf((*pkg_.RuntimeStorageAPI)(nil)).Elem()},
		
		{ "SyncStateAPI", reflect.TypeOf((*pkg_.SyncStateAPI)(nil)).Elem()},
		
	}
	pkg := &model.Package{
		// NOTE: This behaves contrary to documented behaviour if the
		// package name is not the final component of the import path.
		// The reflect package doesn't expose the package name, though.
		Name: path.Base("github.com/ChainSafe/gossamer/dot/rpc/modules"),
	}

	for _, it := range its {
		intf, err := model.InterfaceFromInterfaceType(it.typ)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Reflection: %v\n", err)
			os.Exit(1)
		}
		intf.Name = it.sym
		pkg.Interfaces = append(pkg.Interfaces, intf)
	}

	outfile := os.Stdout
	if len(*output) != 0 {
		var err error
		outfile, err = os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open output file %q", *output)
		}
		defer func() {
			if err := outfile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to close output file %q", *output)
				os.Exit(1)
			}
		}()
	}

	if err := gob.NewEncoder(outfile).Encode(pkg); err != nil {
		fmt.Fprintf(os.Stderr, "gob encode: %v\n", err)
		os.Exit(1)
	}
}
