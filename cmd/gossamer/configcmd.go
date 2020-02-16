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
	log "github.com/ChainSafe/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
)

// buildConfig loads the default gossamer config, then updates the config based
// on the toml configuration file (if provided), then updates the config based
// on any command options (if provided), returning the updated config
func buildConfig(ctx *cli.Context) (*config.Config, error) {

	// load default configuration
	cfg := config.DefaultConfig()

	log.Debug(
		"Set default \"Global\" configuration...",
		"RootDir", cfg.Global.RootDir,
		"Node", cfg.Global.Node,
		"NodeDir", cfg.Global.NodeDir,
	)

	log.Debug(
		"Set default \"Node\" configuration...",
		"Roles", cfg.Node.Roles,
		"Authority", cfg.Node.Authority,
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
		"Host", cfg.RPC.Host,
		"Port", cfg.RPC.Port,
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
	setNodeConfig(ctx, &cfg.Node)
	setNetworkConfig(ctx, &cfg.Network)
	setRPCConfig(ctx, &cfg.RPC)

	return cfg, nil
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

// --root --node
func setGlobalConfig(ctx *cli.Context, cfg *config.GlobalConfig) {

	rootDir := ""
	node := ""

	// --root
	if rootDir = ctx.String(RootDirFlag.Name); rootDir != "" {
		expandedRootDir := expandPath(rootDir)
		cfg.RootDir, _ = filepath.Abs(expandedRootDir)
		log.Debug(
			"Updated configuration...",
			"RootDir", cfg.RootDir,
		)
	}

	// --node
	if node = ctx.String(NodeFlag.Name); node != "" {
		cfg.Node = node
		log.Debug(
			"Updated configuration...",
			"Node", cfg.Node,
		)
	}

	if rootDir != "" || node != "" {
		// create node directory from root directory and node name
		cfg.NodeDir = filepath.Join(cfg.RootDir, cfg.Node)
		log.Debug(
			"Updated configuration...",
			"NodeDir", cfg.NodeDir,
		)
	}

}

// --roles --authority
func setNodeConfig(ctx *cli.Context, cfg *config.NodeConfig) {

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

// --bootnodes --protocol --port --nobootstrap --nomdns
func setNetworkConfig(ctx *cli.Context, cfg *config.NetworkConfig) {

	// --bootnodes
	if bnodes := ctx.GlobalString(BootnodesFlag.Name); bnodes != "" {
		cfg.Bootnodes = strings.Split(ctx.GlobalString(BootnodesFlag.Name), ",")
	}

	// --protocol
	if protocol := ctx.GlobalString(ProtocolIDFlag.Name); protocol != "" {
		cfg.ProtocolID = protocol
	}

	// --port
	if port := ctx.GlobalUint(PortFlag.Name); port != 0 {
		cfg.Port = uint32(port)
	}

	// --nobootstrap
	if off := ctx.GlobalBool(NoBootstrapFlag.Name); off {
		cfg.NoBootstrap = true
	}

	// --nomdns
	if off := ctx.GlobalBool(NoMDNSFlag.Name); off {
		cfg.NoMDNS = true
	}

}

// --rpc --rpcmods --rpchost --rpcport
func setRPCConfig(ctx *cli.Context, cfg *config.RPCConfig) {

	// --rpcmods
	if mods := ctx.GlobalString(RPCModuleFlag.Name); mods != "" {
		cfg.Modules = strToMods(strings.Split(ctx.GlobalString(RPCModuleFlag.Name), ","))
	}

	// --rpchost
	if host := ctx.GlobalString(RPCHostFlag.Name); host != "" {
		cfg.Host = host
	}

	// --rpcport
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
