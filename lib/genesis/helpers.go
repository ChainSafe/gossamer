// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package genesis

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type accountAddr [32]byte

const (
	societyConst             = "Society"
	stakingConst             = "Staking"
	contractsConst           = "Contracts"
	sessionConst             = "Session"
	instance1CollectiveConst = "Instance1Collective"
	instance2CollectiveConst = "Instance2Collective"
	instance1MembershipConst = "Instance1Membership"
	phragmenElectionConst    = "PhragmenElection"
	notForcing               = "NotForcing"
	forceNew                 = "ForceNew"
	forceNone                = "ForceNone"
	forceAlways              = "ForceAlways"
	currentSchedule          = "CurrentSchedule"
	phantom                  = "Phantom"
)

// NewGenesisFromJSONRaw parses a JSON formatted genesis file
func NewGenesisFromJSONRaw(file string) (*Genesis, error) {
	fp, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	g := new(Genesis)
	err = json.Unmarshal(data, g)
	return g, err
}

// trimGenesisAuthority iterates over authorities in genesis and keeps only `authCount` number of authorities.
func trimGenesisAuthority(g *Genesis, authCount int) {
	for k, authMap := range g.Genesis.Runtime {
		if k != "Babe" && k != "Grandpa" {
			continue
		}
		authorities, _ := authMap["Authorities"].([]interface{})
		var newAuthorities []interface{}
		for _, authority := range authorities {
			if len(newAuthorities) >= authCount {
				break
			}
			newAuthorities = append(newAuthorities, authority)
		}
		authMap["Authorities"] = newAuthorities
	}
}

// NewGenesisFromJSON parses Human Readable JSON formatted genesis file.Name. If authCount > 0,
// then it keeps only `authCount` number of authorities for babe and grandpa.
func NewGenesisFromJSON(file string, authCount int) (*Genesis, error) {
	g, err := NewGenesisSpecFromJSON(file)
	if err != nil {
		return nil, err
	}

	if authCount > 0 {
		trimGenesisAuthority(g, authCount)
	}

	grt := g.Genesis.Runtime
	res, err := buildRawMap(grt)
	if err != nil {
		return nil, err
	}

	g.Genesis.Raw = make(map[string]map[string]string)
	g.Genesis.Raw["top"] = res

	return g, err
}

// NewGenesisSpecFromJSON returns a new Genesis (without raw fields) from a human-readable genesis file
func NewGenesisSpecFromJSON(file string) (*Genesis, error) {
	fp, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, err
	}

	g := new(Genesis)

	err = json.Unmarshal(data, g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

// keyValue struct to hold data regarding entry
type keyValue struct {
	key      []string
	value    string
	valueLen *big.Int
	iVal     []interface{}
}

func generatePalletKeyValue(k string, v map[string]interface{}, res map[string]string) (bool, error) {
	jsonBody, err := json.Marshal(v)
	if err != nil {
		return false, err
	}

	var s interface{}
	switch k {
	case societyConst:
		s = &society{}
	case stakingConst:
		s = &staking{}
	case contractsConst:
		c := &contracts{}
		if err = json.Unmarshal(jsonBody, c); err != nil {
			return false, err
		}

		err = generateContractKeyValue(c, k, res)
		if err != nil {
			return false, err
		}
		return true, nil
	case sessionConst:
		sc := &session{}
		if err = json.Unmarshal(jsonBody, sc); err != nil {
			return false, err
		}

		err = generateSessionKeyValue(sc, k, res)
		if err != nil {
			return false, err
		}
		return true, nil
	case instance1CollectiveConst:
		s = &instance1Collective{}
	case instance2CollectiveConst:
		s = &instance2Collective{}
	case instance1MembershipConst:
		s = &instance1Membership{}
	case phragmenElectionConst:
		s = &phragmenElection{}
	default:
		return false, nil
	}
	if err = json.Unmarshal(jsonBody, s); err != nil {
		return false, err
	}
	err = generateKeyValue(s, k, res)

	if err != nil {
		return false, err
	}
	return true, nil
}

func buildRawMap(m map[string]map[string]interface{}) (map[string]string, error) {
	res := make(map[string]string)
	for k, v := range m {
		kv := new(keyValue)
		kv.key = append(kv.key, k)

		ok, err := generatePalletKeyValue(k, v, res)
		if err != nil {
			return nil, err
		}

		if ok {
			continue
		}

		if err = buildRawMapInterface(v, kv); err != nil {
			return nil, err
		}

		if reflect.DeepEqual([]string{"Balances", "balances"}, kv.key) {
			err = buildBalances(kv, res)
			if err != nil {
				return nil, err
			}
			continue
		}

		key, err := formatKey(kv)
		if err != nil {
			return nil, err
		}

		value, err := formatValue(kv)
		if err != nil {
			return nil, err
		}
		res[key] = value
	}

	res[common.BytesToHex(common.UpgradedToDualRefKey)] = "0x01"
	return res, nil
}

func buildRawMapInterface(m map[string]interface{}, kv *keyValue) error {
	for k, v := range m {
		kv.key = append(kv.key, k)
		switch v2 := v.(type) {
		case []interface{}:
			kv.valueLen = big.NewInt(int64(len(v2)))
			if err := buildRawArrayInterface(v2, kv); err != nil {
				return err
			}
		case string:
			kv.value = v2
		}
	}
	return nil
}

func buildRawArrayInterface(a []interface{}, kv *keyValue) error {
	for _, v := range a {
		switch v2 := v.(type) {
		case []interface{}:
			err := buildRawArrayInterface(v2, kv)
			if err != nil {
				return err
			}
		case string:
			// TODO: check to confirm it's an address (#1865)
			tba := crypto.PublicAddressToByteArray(common.Address(v2))
			kv.value = kv.value + fmt.Sprintf("%x", tba)
			kv.iVal = append(kv.iVal, tba)
		case float64:
			encVal, err := scale.Marshal(uint64(v2))
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(int64(v2)))
		}
	}
	return nil
}

func generateStorageKey(modulePrefix, storageKey string) (string, error) {
	moduleName, err := common.Twox128Hash([]byte(modulePrefix))
	if err != nil {
		return "", err
	}

	storagePrefix, err := common.Twox128Hash([]byte(storageKey))
	if err != nil {
		return "", err
	}
	return common.BytesToHex(append(moduleName, storagePrefix...)), nil
}

func generateStorageValue(i interface{}, idx int) ([]byte, error) {
	val := reflect.ValueOf(i)
	var (
		encode []byte
		err    error
	)

	switch t := reflect.Indirect(val).Field(idx).Interface().(type) {
	case int64, uint64, uint32, *scale.Uint128:
		encode, err = scale.Marshal(t)
		if err != nil {
			return nil, err
		}
	case []interface{}:
		enc := make([][32]byte, len(t))
		var accAddr accountAddr
		for index, add := range t {
			copy(accAddr[:], crypto.PublicAddressToByteArray(common.Address(add.(string))))
			enc[index] = accAddr
		}
		encode, err = scale.Marshal(enc)
		if err != nil {
			return nil, err
		}
	case []string:
		for _, add := range t {
			accID := crypto.PublicAddressToByteArray(common.Address(add))
			encode = append(encode, accID...)
		}
		encode, err = scale.Marshal(encode)
		if err != nil {
			return nil, err
		}
	case string:
		var value uint8
		switch t {
		case notForcing:
			value = 0
		case forceNew:
			value = 1
		case forceNone:
			value = 2
		case forceAlways:
			value = 3
		}
		encode, err = scale.Marshal(value)
		if err != nil {
			return nil, err
		}
	case [][]interface{}:
		// TODO: for members field in phragmenElection struct figure out the correct format for encoding value (#1866)
		for _, data := range t {
			for _, v := range data {
				var accAddr accountAddr
				switch v1 := v.(type) {
				case string:
					copy(accAddr[:], crypto.PublicAddressToByteArray(common.Address(v1)))
					encode = append(encode, accAddr[:]...)
				case float64:
					var bytesVal []byte
					bytesVal, err = scale.Marshal(big.NewInt(int64(v1)))
					if err != nil {
						return nil, err
					}
					encode = append(encode, bytesVal...)
				}
			}
		}

		encode, err = scale.Marshal(encode)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid value type")
	}
	return encode, nil
}

func generateContractKeyValue(c *contracts, prefixKey string, res map[string]string) error {
	var (
		key string
		err error
	)

	// First field of contract is the storage key
	val := reflect.ValueOf(c)
	if k := reflect.Indirect(val).Type().Field(0).Name; k == currentSchedule {
		key, err = generateStorageKey(prefixKey, k)
		if err != nil {
			return err
		}
	}

	encode, err := scale.Marshal(c)
	if err != nil {
		return err
	}

	res[key] = common.BytesToHex(encode)
	return nil
}

func generateKeyValue(s interface{}, prefixKey string, res map[string]string) error {
	val := reflect.ValueOf(s)
	n := reflect.Indirect(val).NumField()

	for i := 0; i < n; i++ {
		val := reflect.ValueOf(s)
		storageKey := reflect.Indirect(val).Type().Field(i).Name
		if storageKey == phantom { // ignore Phantom as its value is null
			continue
		}

		key, err := generateStorageKey(prefixKey, storageKey)
		if err != nil {
			return err
		}

		value, err := generateStorageValue(s, i)
		if err != nil {
			return err
		}

		res[key] = common.BytesToHex(value)
	}
	return nil
}

func formatKey(kv *keyValue) (string, error) {
	switch {
	case reflect.DeepEqual([]string{"Grandpa", "Authorities"}, kv.key):
		kb := []byte(`:grandpa_authorities`)
		return common.BytesToHex(kb), nil
	case reflect.DeepEqual([]string{"System", "code"}, kv.key):
		kb := []byte(`:code`)
		return common.BytesToHex(kb), nil
	default:
		if len(kv.key) < 2 {
			return "", errors.New("key array less than 2")
		}

		return generateStorageKey(kv.key[0], kv.key[1])
	}
}

func generateSessionKeyValue(s *session, prefixKey string, res map[string]string) error {
	val := reflect.ValueOf(s).Elem()
	moduleName, err := common.Twox128Hash([]byte(prefixKey))
	if err != nil {
		return err
	}

	storageVal, ok := val.Field(0).Interface().([][]interface{})
	if !ok {
		return nil
	}

	for _, strV := range storageVal {
		for _, v := range strV {
			var validatorAccID []byte
			switch t := v.(type) {
			case string:
				var nextKeyHash []byte
				nextKeyHash, err = common.Twox128Hash([]byte("NextKeys"))
				if err != nil {
					return err
				}

				validatorAccID = crypto.PublicAddressToByteArray(common.Address(t))
				var accIDHash []byte
				accIDHash, err = common.Twox64(validatorAccID)
				if err != nil {
					return err
				}

				prefix := bytes.Join([][]byte{moduleName, nextKeyHash}, nil)
				suffix := bytes.Join([][]byte{accIDHash, validatorAccID}, nil)
				res[common.BytesToHex(append(prefix, suffix...))] = common.BytesToHex(validatorAccID)
			case map[string]interface{}:
				var storagePrefixKey []byte
				storagePrefixKey, err = common.Twox128Hash([]byte("KeyOwner"))
				if err != nil {
					return err
				}

				storagePrefixKey = append(moduleName, storagePrefixKey...)
				for key, v1 := range t {
					var addressKey []byte
					switch key {
					case "grandpa":
						addressKey, err = generateAddressHash(v1.(string), "gran")
						if err != nil {
							return err
						}
					case "babe":
						addressKey, err = generateAddressHash(v1.(string), "babe")
						if err != nil {
							return err
						}
					case "im_online":
						addressKey, err = generateAddressHash(v1.(string), "imon")
						if err != nil {
							return err
						}
					case "authority_discovery":
						addressKey, err = generateAddressHash(v1.(string), "audi")
						if err != nil {
							return err
						}
					default:
						return fmt.Errorf("invalid storage keys")
					}

					res[common.BytesToHex(append(storagePrefixKey, addressKey...))] = common.BytesToHex(validatorAccID)
				}
			}
		}
	}

	return nil
}

func generateAddressHash(accAddr, key string) ([]byte, error) {
	acc := crypto.PublicAddressToByteArray(common.Address(accAddr))
	encodeAcc, _ := scale.Marshal(acc)
	storageKey := append([]byte(key), encodeAcc...)
	addersHash, err := common.Twox64(storageKey)
	if err != nil {
		return nil, err
	}

	return append(addersHash, storageKey...), err
}
func formatValue(kv *keyValue) (string, error) {
	switch {
	case reflect.DeepEqual([]string{"Grandpa", "Authorities"}, kv.key):
		if kv.valueLen != nil {
			lenEnc, err := scale.Marshal(kv.valueLen)
			if err != nil {
				return "", err
			}
			// prepend 01 to grandpa_authorities values
			return fmt.Sprintf("0x01%x%v", lenEnc, kv.value), nil
		}
		return "", fmt.Errorf("error formatting value for grandpa authorities")
	case reflect.DeepEqual([]string{"System", "code"}, kv.key):
		return kv.value, nil
	case reflect.DeepEqual([]string{"Sudo", "Key"}, kv.key):
		return common.BytesToHex(crypto.PublicAddressToByteArray(common.Address(kv.value))), nil
	default:
		if kv.valueLen != nil {
			lenEnc, err := scale.Marshal(kv.valueLen)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("0x%x%v", lenEnc, kv.value), nil
		}
		return fmt.Sprintf("0x%x", kv.value), nil
	}
}

func buildBalances(kv *keyValue, res map[string]string) error {
	for i := range kv.iVal {
		if i%2 == 0 {
			// build key
			bKey := common.MustHexToBytes(systemAccountKeyHex)

			addHash, err := common.Blake2b128(kv.iVal[i].([]byte))
			if err != nil {
				return err
			}
			bKey = append(bKey, addHash...)

			bKey = append(bKey, kv.iVal[i].([]byte)...)

			accInfo := types.AccountInfo{
				Nonce: 0,
				//RefCount: 0,
				Data: types.AccountData{
					Free:       scale.MustNewUint128(kv.iVal[i+1].(*big.Int)),
					Reserved:   scale.MustNewUint128(big.NewInt(0)),
					MiscFrozen: scale.MustNewUint128(big.NewInt(0)),
					FreeFrozen: scale.MustNewUint128(big.NewInt(0)),
				},
			}

			encBal, err := scale.Marshal(accInfo)
			if err != nil {
				return err
			}
			res[common.BytesToHex(bKey)] = common.BytesToHex(encBal)
		}
	}
	return nil
}

// BuildFromMap builds genesis fields data from map
func BuildFromMap(m map[string][]byte, gen *Genesis) error {
	for k, v := range m {
		key := fmt.Sprintf("0x%x", k)
		switch key {

		case "0x3a636f6465":
			// handle :code
			addCodeValue(v, gen)
			addRawValue(key, v, gen)
		case GrandpaAuthoritiesKeyHex:
			// handle :grandpa_authorities
			//  slice value since it was encoded starting with 0x01
			err := addAuthoritiesValues("grandpa", "authorities", crypto.Ed25519Type, v[1:], gen)
			if err != nil {
				return err
			}
			addRawValue(key, v, gen)
		case BABEAuthoritiesKeyHex:
			// handle Babe Authorities
			err := addAuthoritiesValues("babe", "authorities", crypto.Sr25519Type, v, gen)
			if err != nil {
				return err
			}
			addRawValue(key, v, gen)
		}
	}
	return nil
}

func addRawValue(key string, value []byte, gen *Genesis) {
	if gen.Genesis.Raw["top"] == nil {
		gen.Genesis.Raw["top"] = make(map[string]string)
	}
	gen.Genesis.Raw["top"][key] = common.BytesToHex(value)
}

func addCodeValue(value []byte, gen *Genesis) {
	if gen.Genesis.Runtime["system"] == nil {
		gen.Genesis.Runtime["system"] = make(map[string]interface{})
	}
	gen.Genesis.Runtime["system"]["code"] = common.BytesToHex(value)
}

func addAuthoritiesValues(k1, k2 string, kt crypto.KeyType, value []byte, gen *Genesis) error {
	if gen.Genesis.Runtime[k1] == nil {
		gen.Genesis.Runtime[k1] = make(map[string]interface{})
	}

	var auths []types.AuthorityRaw
	err := scale.Unmarshal(value, &auths)
	if err != nil {
		return err
	}

	authAddrs, err := types.AuthoritiesRawToAuthorityAsAddress(auths, kt)
	if err != nil {
		return err
	}

	gen.Genesis.Runtime[k1][k2] = authAddrs
	return nil
}
