package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ChainSafe/gossamer/cmd/utils"
	"github.com/ChainSafe/gossamer/crypto"
	"github.com/ChainSafe/gossamer/keystore"

	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
)

func handleAccounts(ctx *cli.Context) {
	err := startLogger(ctx)
	if err != nil {
		log.Error("account", "error", err)
	}

	if keygen := ctx.Bool(utils.GenerateFlag.Name); keygen {
		log.Info("generating keypair...")
		generateKeypair(ctx)
	}

	if keyimport := ctx.String(utils.ImportFlag.Name); keyimport != "" {
		// TODO: import keys from encrypted file
		log.Info("importing keypair...")
	}

	if keylist := ctx.Bool(utils.ListFlag.Name); keylist {
		// list all keys
		listKeys()
	}
}

func listKeys() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Error("list error: could not get home dir", "err", err)
		os.Exit(1)
	}

	keystorepath, err := filepath.Abs(home + "/.gossamer/keystore/")
	if err != nil {
		log.Error("list error: could not get keystore dir", "err", err)
		os.Exit(1)
	}

	files, err := ioutil.ReadDir(keystorepath)
	if err != nil {
		log.Error("list error: could not read keystore dir", "err", err)
		os.Exit(1)
	}

	for _, f := range files {
		fmt.Println(f.Name())
	}
}

func generateKeypair(ctx *cli.Context) {
	keytype := ""

	// check if --type flag is set
	if flagtype := ctx.String(utils.AccountTypeFlag.Name); flagtype != "" {
		// check if keytype is ed25519 or sr25519
		if flagtype == "sr25519" || flagtype == "ed25519" {
			keytype = flagtype
		} else {
			log.Error("generate error: invalid type supplied; must be sr25519 or ed25519", "type", flagtype)
			os.Exit(1)
		}
	}

	password := getPassword()

	if keytype == "" {
		keytype = "sr25519"
	}

	var kp crypto.Keypair
	var err error
	if keytype == "sr25519" {
		// generate sr25519 keys
		kp, err = crypto.GenerateSr25519Keypair()
		if err != nil {
			log.Error("generate error: could not generate sr25519 keypair", "err", err)
			os.Exit(1)
		}
	} else if keytype == "ed25519" {
		// generate ed25519 keys
		kp, err = crypto.GenerateEd25519Keypair()
		if err != nil {
			log.Error("generate error: could not generate ed25519 keypair", "err", err)
			os.Exit(1)
		}
	}

	filename := hex.EncodeToString(kp.Public().Encode())
	home, err := os.UserHomeDir()
	if err != nil {
		log.Error("generate error: could not get home dir", "err", err)
		os.Exit(1)
	}

	fp, err := filepath.Abs(home + "/.gossamer/keystore/" + filename + ".key")
	if err != nil {
		log.Error("generate error: invalid filepath", "err", err)
		os.Exit(1)
	}

	keystorepath, err := filepath.Abs(home + "/.gossamer/keystore/")
	if _, err := os.Stat(keystorepath); os.IsNotExist(err) {
		os.Mkdir(keystorepath, os.ModePerm)
	}

	file, err := os.OpenFile(fp, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error("", "err", err)
		os.Exit(1)
	}

	err = keystore.EncryptAndWriteToFile(file, kp.Private(), password)
	if err != nil {
		log.Error("generate error: could not write key to file", "err", err)
	}

	log.Info("key generated", "public key", filename, "file", fp)
}

func getPassword() []byte {
	buf := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Enter password to encrypt keystore file:")
		fmt.Print("> ")
		password, err := buf.ReadBytes('\n')
		if err != nil {
			fmt.Printf("invalid input: %s\n", err)
		} else {
			return password
		}
	}
}
