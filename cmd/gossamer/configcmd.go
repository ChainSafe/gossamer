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

package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/internal/api"
	"github.com/ChainSafe/gossamer/keystore"
	log "github.com/ChainSafe/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

// buildConfig updates initialized configuration from flags
func buildConfig(ctx *cli.Context) (*config.Config, error) {

	// load default configuration
	cfg := config.DefaultConfig()

	log.Debug(
		"Set default \"Global\" configuration...",
		"DataDir", cfg.Global.DataDir,
		"Chain", cfg.Global.Chain,
		"Roles", cfg.Global.Roles,
		"Authority", cfg.Global.Authority,
	)

	log.Debug(
		"Set default \"Network\" configuration...",
		"Bootnodes", cfg.Network.Bootnodes,
		"ProtocolID", cfg.Network.ProtocolID,
		"Port", cfg.Network.Port,
		"NoBootstrap", cfg.Network.NoBootstrap,
		"NoMDNS", cfg.Network.NoMDNS,
	)

	log.Debug(
		"Set default \"RPC\" configuration...",
		"Port", cfg.RPC.Port,
		"Host", cfg.RPC.Host,
		"Modules", cfg.RPC.Modules,
	)

	// --config
	if filename := ctx.GlobalString(ConfigFileFlag.Name); filename != "" {
		log.Debug(
			"Loading toml configuration file...",
			"filename", filename,
		)
		err := loadConfig(filename, cfg)
		if err != nil {
			log.Error("Failed to load toml configuration file", "err", err)
			return nil, err
		}
	}

	// parse flags and update configuration
	setGlobalConfig(ctx, &cfg.Global)
	setNetworkConfig(ctx, &cfg.Network)
	setRPCConfig(ctx, &cfg.RPC)

	return cfg, nil
}

// unlockAccount
func unlockAccount(ctx *cli.Context, cfg *config.Config) (*keystore.Keystore, error) {
	// --unlock - load all static keys from keystore directory
	ks := keystore.NewKeystore()
	// unlock keys, if specified
	if keyindices := ctx.String(UnlockFlag.Name); keyindices != "" {
		err := unlockKeys(ctx, cfg.Global.DataDir, ks)
		if err != nil {
			return nil, fmt.Errorf("could not unlock keys: %s", err)
		}
	}
	return ks, nil
}

// loadConfig loads the contents from config toml and inits Config object
func loadConfig(file string, cfg *config.Config) error {
	fp, err := filepath.Abs(file)
	if err != nil {
		return err
	}
	f, err := os.Open(filepath.Clean(fp))
	if err != nil {
		return err
	}
	if err = tomlSettings.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}
	return nil
}

// --config --datadir --roles
func setGlobalConfig(ctx *cli.Context, cfg *config.GlobalConfig) {

	// --datadir
	if dataDir := ctx.String(DataDirFlag.Name); dataDir != "" {
		expandedDataDir := expandPath(dataDir)
		cfg.DataDir, _ = filepath.Abs(expandedDataDir)
		log.Debug(
			"Updated configuration...",
			"DataDir", cfg.DataDir,
		)
	}

	// --chain
	if chain := ctx.String(ChainFlag.Name); chain != "" {
		cfg.Chain = chain
		log.Debug(
			"Updated configuration...",
			"Chain", cfg.Chain,
		)
	}

	// --roles
	if roles := ctx.GlobalString(RolesFlag.Name); roles != "" {
		b, err := strconv.Atoi(roles)
		if err != nil {
			log.Error(
				"Failed to convert string to byte",
				"Roles", roles,
			)
		} else {
			cfg.Roles = byte(b)
			log.Debug(
				"Updated configuration...",
				"Roles", cfg.Roles,
			)
		}
	}

	// --authority
	if auth := ctx.GlobalBool(AuthorityFlag.Name); auth && !cfg.Authority {
		cfg.Authority = true
		log.Debug(
			"Updated configuration...",
			"Authority", cfg.Authority,
		)
	} else if ctx.IsSet(AuthorityFlag.Name) && !auth && cfg.Authority {
		cfg.Authority = false
		log.Debug(
			"Updated configuration...",
			"Authority", cfg.Authority,
		)
	}
}

func setNetworkConfig(ctx *cli.Context, cfg *config.NetworkCfg) {
	// Bootnodes
	if bnodes := ctx.GlobalString(BootnodesFlag.Name); bnodes != "" {
		cfg.Bootnodes = strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",")
	}

	if protocol := ctx.GlobalString(ProtocolIDFlag.Name); protocol != "" {
		cfg.ProtocolID = protocol
	}

	if port := ctx.GlobalUint(PortFlag.Name); port != 0 {
		cfg.Port = uint32(port)
	}

	// NoBootstrap
	if off := ctx.GlobalBool(NoBootstrapFlag.Name); off {
		cfg.NoBootstrap = true
	}

	// NoMDNS
	if off := ctx.GlobalBool(NoMDNSFlag.Name); off {
		cfg.NoMDNS = true
	}
}

func setRPCConfig(ctx *cli.Context, cfg *config.RPCCfg) {
	// Modules
	if mods := ctx.GlobalString(RPCModuleFlag.Name); mods != "" {
		cfg.Modules = strToMods(strings.Split(ctx.GlobalString(RPCModuleFlag.Name), ","))
	}

	// Host
	if host := ctx.GlobalString(RPCHostFlag.Name); host != "" {
		cfg.Host = host
	}

	// Port
	if port := ctx.GlobalUint(RPCPortFlag.Name); port != 0 {
		cfg.Port = uint32(port)
	}

}

// strToMods casts a []strings to []api.Module
func strToMods(strs []string) []api.Module {
	var res []api.Module
	for _, str := range strs {
		res = append(res, api.Module(str))
	}
	return res
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	cfg, err := buildConfig(ctx)
	if err != nil {
		return err
	}

	comment := ""

	out, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		/* #nosec */
		dump, err = os.OpenFile(filepath.Clean(ctx.Args().Get(0)), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}

		defer func() {
			err = dump.Close()
			if err != nil {
				log.Warn("err closing conn", "err", err.Error())
			}
		}()
	}
	_, err = dump.WriteString(comment)
	if err != nil {
		log.Warn("err writing comment output for dumpconfig command", "err", err.Error())
	}
	_, err = dump.Write(out)
	if err != nil {
		log.Warn("err writing comment output for dumpconfig command", "err", err.Error())
	}
	return nil
}

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

// expandPath will expand a tilde prefix path to full home path
func expandPath(targetPath string) string {
	if strings.HasPrefix(targetPath, "~\\") || strings.HasPrefix(targetPath, "~/") {
		if homeDir := config.HomeDir(); homeDir != "" {
			targetPath = homeDir + targetPath[1:]
		}
	} else if strings.HasPrefix(targetPath, ".\\") || strings.HasPrefix(targetPath, "./") {
		targetPath, _ = filepath.Abs(targetPath)
	}
	return path.Clean(os.ExpandEnv(targetPath))
}
