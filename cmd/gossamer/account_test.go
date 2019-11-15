package main

import (
	"os"
	"testing"
)

var testKeystoreDir = "./test_keystore"

func TestGenerateKey(t *testing.T) {
	ctx, err := createCliContext("account generate", []string{"generate", "datadir"}, []interface{}{true, testKeystoreDir})
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(testKeystoreDir)

	command := accountCommand
	err = command.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

// func TestImportKey(t *testing.T) {
// 	filename := ""
// 	defer os.RemoveAll(testKeystoreDir)

// 	ctx, err := createCliContext("account import", []string{"import", "datadir"}, []interface{}{filename, testKeystoreDir})
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	command := accountCommand
// 	err = command.Run(ctx)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// }
