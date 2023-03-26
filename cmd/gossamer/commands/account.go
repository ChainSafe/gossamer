package commands

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/utils"
	"github.com/spf13/cobra"
)

func init() {
	accountCmd.Flags().String("keystore-path", "", "path to keystore")
	accountCmd.Flags().String("keystore-file", "", "name of keystore file to import")
	accountCmd.Flags().String("password", "", "password used to encrypt the keystore. Used with --generate or --unlock")
	accountCmd.Flags().String("scheme", crypto.Sr25519Type, "keyring scheme (sr25519, ed25519, secp256k1)")
}

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "account",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			logger.Errorf("account command cannot be empty")
			return cmd.Help()
		}

		switch args[0] {
		case "generate":
			if err := generateKeyPair(cmd); err != nil {
				return err
			}
		case "import":
			if err := importKey(cmd); err != nil {
				return err
			}
		case "import-raw":
			if err := importRawKey(cmd); err != nil {
				return err
			}
		case "list":
			if err := listKeys(cmd); err != nil {
				return err
			}
		default:
			logger.Errorf("invalid account command: %s", args[0])
			cmd.Help()
			return fmt.Errorf("invalid account command: %s", args[0])
		}

		return nil
	},
}

func generateKeyPair(cmd *cobra.Command) error {
	keystorePath, err := cmd.Flags().GetString("keystore-path")
	if err != nil {
		return fmt.Errorf("failed to get keystore-path: %s", err)
	}
	if keystorePath == "" {
		return fmt.Errorf("keystore-path cannot be empty")
	}

	scheme, err := cmd.Flags().GetString("scheme")
	if err != nil {
		return fmt.Errorf("failed to get scheme: %s", err)
	}
	if !(scheme == crypto.Ed25519Type || scheme == crypto.Sr25519Type || scheme == crypto.Secp256k1Type) {
		return fmt.Errorf("invalid scheme: %s", scheme)
	}

	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return fmt.Errorf("failed to get password: %s", err)
	}

	logger.Info("Generating keypair")

	file, err := keystore.GenerateKeypair(scheme, nil, keystorePath, []byte(password))
	if err != nil {
		logger.Errorf("failed to generate keypair: %s", err)
		return err
	}

	logger.Infof("keypair generated and saved to %s", file)

	return nil
}

func importKey(cmd *cobra.Command) error {
	keystorePath, err := cmd.Flags().GetString("keystore-path")
	if err != nil {
		return fmt.Errorf("failed to get keystore-path: %s", err)
	}
	if keystorePath == "" {
		return fmt.Errorf("keystore-path cannot be empty")
	}

	keystoreFile, err := cmd.Flags().GetString("keystore-file")
	if err != nil {
		return fmt.Errorf("failed to get keystore-file: %s", err)
	}
	if keystoreFile == "" {
		return fmt.Errorf("keystore-file cannot be empty")
	}

	_, err = keystore.ImportKeypair(keystoreFile, keystorePath)
	if err != nil {
		logger.Errorf("failed to import keypair: %s", err)
		return err
	}

	return nil
}

func importRawKey(cmd *cobra.Command) error {
	keystorePath, err := cmd.Flags().GetString("keystore-path")
	if err != nil {
		return fmt.Errorf("failed to get keystore-path: %s", err)
	}
	if keystorePath == "" {
		return fmt.Errorf("keystore-path cannot be empty")
	}

	keystoreFile, err := cmd.Flags().GetString("keystore-file")
	if err != nil {
		return fmt.Errorf("failed to get keystore-file: %s", err)
	}
	if keystoreFile == "" {
		return fmt.Errorf("keystore-file cannot be empty")
	}

	scheme, err := cmd.Flags().GetString("scheme")
	if err != nil {
		return fmt.Errorf("failed to get scheme: %s", err)
	}
	if !(scheme == crypto.Ed25519Type || scheme == crypto.Sr25519Type || scheme == crypto.Secp256k1Type) {
		return fmt.Errorf("invalid scheme: %s", scheme)
	}

	password, err := cmd.Flags().GetString("password")
	if err != nil {
		return fmt.Errorf("failed to get password: %s", err)
	}

	file, err := keystore.ImportRawPrivateKey(keystoreFile, scheme, keystorePath, []byte(password))
	if err != nil {
		logger.Errorf("failed to import private key: %s", err)
		return err
	}

	logger.Info("imported private key and saved it to " + file)

	return nil
}

func listKeys(cmd *cobra.Command) error {
	keystorePath, err := cmd.Flags().GetString("keystore-path")
	if err != nil {
		return fmt.Errorf("failed to get keystore-path: %s", err)
	}
	if keystorePath == "" {
		return fmt.Errorf("keystore-path cannot be empty")
	}

	_, err = utils.KeystoreFilepaths(keystorePath)
	if err != nil {
		logger.Errorf("failed to list keys: %s", err)
		return err
	}

	return nil
}
