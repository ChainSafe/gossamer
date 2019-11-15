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

	var datadir string
	if dir := ctx.String(utils.DataDirFlag.Name); dir != "" {
		datadir, err = filepath.Abs(dir)
		if err != nil {
			log.Error("invalid datadir", "error", err)
			os.Exit(1)
		}
	}

	if keygen := ctx.Bool(utils.GenerateFlag.Name); keygen {
		log.Info("generating keypair...")

		keytype := ""
		// check if --type flag is set
		if flagtype := ctx.String(utils.AccountTypeFlag.Name); flagtype != "" {
			// check if keytype is ed25519 or sr25519
			if flagtype == "sr25519" || flagtype == "ed25519" {
				keytype = flagtype
			} else {
				log.Error("invalid type supplied; must be sr25519 or ed25519", "type", err)
				os.Exit(1)
			}
		}

		_, err = generateKeypair(keytype, datadir, nil)
		if err != nil {
			log.Error("generate error", "error", err)
			os.Exit(1)
		}
	}

	if keyimport := ctx.String(utils.ImportFlag.Name); keyimport != "" {
		log.Info("importing key...")
		_, err = importKey(keyimport, datadir)
		if err != nil {
			log.Error("import error", "error", err)
			os.Exit(1)
		}
	}

	if keylist := ctx.Bool(utils.ListFlag.Name); keylist {
		_, err = listKeys(datadir)
		if err != nil {
			log.Error("list error", "error", err)
			os.Exit(1)
		}
	}
}

func importKey(filename, datadir string) (string, error) {
	keystorepath, err := keystoreDir(datadir)
	if err != nil {
		return "", fmt.Errorf("could not get keystore directory: %s", err)
	}

	importdata, err := ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		return "", fmt.Errorf("could not read import file: %s", err)
	}

	ksjson := new(keystore.EncryptedKeystore)
	err = json.Unmarshal(importdata, ksjson)
	if err != nil {
		return "", fmt.Errorf("could not read file contents: %s", err)
	}

	keystorefile, err := filepath.Abs(keystorepath + "/" + ksjson.PublicKey[2:] + ".key")
	if err != nil {
		return "", fmt.Errorf("could not create keystore file path: %s", err)
	}

	err = ioutil.WriteFile(keystorefile, importdata, 0644)
	if err != nil {
		return "", fmt.Errorf("could not write to keystore directory: %s", err)
	}

	log.Info("successfully imported key", "public key", ksjson.PublicKey, "file", keystorefile)
	return keystorefile, nil
}

func listKeys(datadir string) ([]string, error) {
	keystorepath, err := keystoreDir(datadir)
	if err != nil {
		return nil, fmt.Errorf("could not get keystore directory: %s", err)
	}

	files, err := ioutil.ReadDir(keystorepath)
	if err != nil {
		return nil, fmt.Errorf("could not read keystore dir: %s", err)
	}

	keys := []string{}

	for _, f := range files {
		fmt.Println(f.Name())
		keys = append(keys, f.Name())
	}

	return keys, nil
}

func generateKeypair(keytype, datadir string, password []byte) (string, error) {
	if password == nil {
		password = getPassword()
	}

	if keytype == "" {
		keytype = "sr25519"
	}

	var kp crypto.Keypair
	var err error
	if keytype == "sr25519" {
		// generate sr25519 keys
		kp, err = crypto.GenerateSr25519Keypair()
		if err != nil {
			return "", fmt.Errorf("could not generate sr25519 keypair: %s", err)
		}
	} else if keytype == "ed25519" {
		// generate ed25519 keys
		kp, err = crypto.GenerateEd25519Keypair()
		if err != nil {
			return "", fmt.Errorf("could not generate ed25519 keypair: %s", err)
		}
	}

	keystorepath, err := keystoreDir(datadir)
	if err != nil {
		return "", fmt.Errorf("could not get keystore directory: %s", err)
	}

	pub := hex.EncodeToString(kp.Public().Encode())
	fp, err := filepath.Abs(keystorepath + "/" + pub + ".key")
	if err != nil {
		return "", fmt.Errorf("invalid filepath: %s", err)
	}

	file, err := os.OpenFile(fp, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}

	defer func() {
		err = file.Close()
		if err != nil {
			log.Error("generate keypair: could not close keystore file")
		}
	}()

	err = keystore.EncryptAndWriteToFile(file, kp.Private(), password)
	if err != nil {
		return "", fmt.Errorf("could not write key to file: %s", err)
	}

	log.Info("key generated", "public key", pub, "file", fp)
	return fp, nil
}

func keystoreDir(datadir string) (string, error) {
	if datadir != "" {
		keystorepath, err := filepath.Abs(datadir)
		if err != nil {
			return "", err
		}

		if _, err := os.Stat(keystorepath); os.IsNotExist(err) {
			err = os.Mkdir(keystorepath, os.ModePerm)
			if err != nil {
				return "", err
			}
		}

		return keystorepath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	keystorepath, err := filepath.Abs(home + "/.gossamer/keystore")
	if err != nil {
		return "", fmt.Errorf("could not create keystore file path: %s", err)
	}

	if _, err := os.Stat(keystorepath); os.IsNotExist(err) {
		err = os.Mkdir(keystorepath, os.ModePerm)
		if err != nil {
			return "", err
		}
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
