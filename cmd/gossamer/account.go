package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/ChainSafe/gossamer/cmd/utils"
	"github.com/ChainSafe/gossamer/crypto"
	"github.com/ChainSafe/gossamer/keystore"

	log "github.com/ChainSafe/log15"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

func handleAccounts(ctx *cli.Context) {
	err := startLogger(ctx)
	if err != nil {
		log.Error("account", "error", err)
	}

	if keygen := ctx.Bool(utils.GenerateFlag.Name); keygen {
		log.Info("generating keypair...")
		err = generateKeypair(ctx)
		if err != nil {
			log.Error("generate error", "error", err)
			os.Exit(1)
		}
	}

	if keyimport := ctx.String(utils.ImportFlag.Name); keyimport != "" {
		log.Info("importing key...")
		err = importKey(keyimport)
		if err != nil {
			log.Error("import error", "error", err)
			os.Exit(1)
		}
	}

	if keylist := ctx.Bool(utils.ListFlag.Name); keylist {
		err = listKeys()
		if err != nil {
			log.Error("list error", "error", err)
			os.Exit(1)
		}
	}
}

func importKey(filename string) error {
	keystorepath, err := keystoreDir()
	if err != nil {
		return fmt.Errorf("could not get keystore directory: %s", err)
	}
	
	importdata, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read import file: %s", err)
	}

	ksjson := new(keystore.EncryptedKeystore)
	err = json.Unmarshal(importdata, ksjson)
	if err != nil {
		return fmt.Errorf("could not read file contents: %s", err)
	}

	keystorefile, err := filepath.Abs(keystorepath + "/" + ksjson.PublicKey[2:] + ".key")

	err = ioutil.WriteFile(keystorefile, importdata, 0644)
	if err != nil {
		fmt.Errorf("could not write to keystore directory: %s", err)
	}

	log.Info("successfully imported key", "public key", ksjson.PublicKey, "file", keystorefile)
	return nil
}

func listKeys() error {
	keystorepath, err := keystoreDir()
	if err != nil {
		return fmt.Errorf("could not get keystore directory: %s", err)
	}
	
	files, err := ioutil.ReadDir(keystorepath)
	if err != nil {
		return fmt.Errorf("could not read keystore dir: %s", err)
	}

	for _, f := range files {
		fmt.Println(f.Name())
	}

	return nil
}

func generateKeypair(ctx *cli.Context) error {
	keytype := ""

	// check if --type flag is set
	if flagtype := ctx.String(utils.AccountTypeFlag.Name); flagtype != "" {
		// check if keytype is ed25519 or sr25519
		if flagtype == "sr25519" || flagtype == "ed25519" {
			keytype = flagtype
		} else {
			return fmt.Errorf("invalid type supplied; must be sr25519 or ed25519: type=%s", flagtype)
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
			return fmt.Errorf("could not generate sr25519 keypair: %s", err)
		}
	} else if keytype == "ed25519" {
		// generate ed25519 keys
		kp, err = crypto.GenerateEd25519Keypair()
		if err != nil {
			return fmt.Errorf("could not generate ed25519 keypair: %s", err)
		}
	}

	keystorepath, err := keystoreDir()
	if err != nil {
		return fmt.Errorf("could not get keystore directory: %s", err)
	}

	filename := hex.EncodeToString(kp.Public().Encode())
	fp, err := filepath.Abs(keystorepath + "/" + filename + ".key")
	if err != nil {
		fmt.Errorf("invalid filepath: %s", err)
	}

	file, err := os.OpenFile(fp, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	defer file.Close()

	err = keystore.EncryptAndWriteToFile(file, kp.Private(), password)
	if err != nil {
		return fmt.Errorf("could not write key to file: %s", err)
	}

	log.Info("key generated", "public key", filename, "file", fp)
	return nil
}

func keystoreDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	keystorepath, err := filepath.Abs(home + "/.gossamer/keystore")
	if _, err := os.Stat(keystorepath); os.IsNotExist(err) {
		os.Mkdir(keystorepath, os.ModePerm)
	}

	return keystorepath, nil
}

func getPassword() []byte {
	for {
		fmt.Println("Enter password to encrypt keystore file:")
		fmt.Print("> ")
		password, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Printf("invalid input: %s\n", err)
		} else {
			return password
		}
	}
}
