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
	"strings"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type accountAddr [32]byte

const (
	stakingConst = "staking"
	sessionConst = "session"
	notForcing   = "NotForcing"
	forceNew     = "ForceNew"
	forceNone    = "ForceNone"
	forceAlways  = "ForceAlways"
	phantom      = "Phantom"
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
	if g.Genesis.Runtime == nil {
		return
	}

	babe := g.Genesis.Runtime.Babe
	if babe != nil {
		if len(babe.Authorities) > authCount {
			babe.Authorities = babe.Authorities[:authCount]
		}
	}

	grandpa := g.Genesis.Runtime.Grandpa
	if grandpa != nil {
		if len(grandpa.Authorities) > authCount {
			grandpa.Authorities = grandpa.Authorities[:authCount]
		}
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

	g.Genesis.Raw = make(map[string]map[string]string)
	grt := g.Genesis.Runtime
	if grt == nil {
		return g, nil
	}

	res, err := buildRawMap(*grt)
	if err != nil {
		return nil, err
	}

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

func generatePalletKeyValue(k string, v interface{}, res map[string]string) (bool, error) {
	jsonBody, err := json.Marshal(v)
	if err != nil {
		return false, err
	}

	var s interface{}
	switch k {
	case stakingConst:
		s = &staking{}

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

func buildRawMap(m Runtime) (map[string]string, error) {
	res := make(map[string]string)
	mRefObjVal := reflect.ValueOf(m)

	for i := 0; i < mRefObjVal.NumField(); i++ {
		v := mRefObjVal.Field(i)
		vInterface := v.Interface()
		if v.IsNil() {
			continue
		}

		jsonTag := mRefObjVal.Type().Field(i).Tag.Get("json")
		k := strings.Split(jsonTag, ",")[0]
		kv := new(keyValue)
		kv.key = append(kv.key, k)

		ok, err := generatePalletKeyValue(k, vInterface, res)
		if err != nil {
			return nil, err
		}

		if ok {
			continue
		}

		if err = buildRawMapInterface(vInterface, kv); err != nil {
			return nil, err
		}

		if reflect.DeepEqual([]string{"balances", "balances"}, kv.key) {
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

func buildRawMapInterface(m interface{}, kv *keyValue) error {
	mRefObjVal := reflect.Indirect(reflect.ValueOf(m))

	for i := 0; i < mRefObjVal.NumField(); i++ {
		jsonTag := mRefObjVal.Type().Field(i).Tag.Get("json")
		k := strings.Split(jsonTag, ",")[0]
		kv.key = append(kv.key, k)
		v := mRefObjVal.Field(i)

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				continue
			}
			v = v.Elem()
		}

		if v.IsZero() {
			continue
		}

		switch v2 := v.Interface().(type) {
		case string:
			kv.value = v2
		case uint64, int64, int:
			kv.value = fmt.Sprint(v2)
		default:

			switch v.Kind() {
			case reflect.Slice:

				vLen := v.Len()
				listOfInterface := []interface{}{}
				for i := 0; i < vLen; i++ {
					listOfInterface = append(listOfInterface, v.Index(i).Interface())
				}

				if vLen > 0 && v.Index(0).Kind() == reflect.Struct {
					kv.valueLen = big.NewInt(int64(v.Index(0).NumField()))
				} else {
					kv.valueLen = big.NewInt(int64(vLen))

				}
				if err := buildRawArrayInterface(listOfInterface, kv); err != nil {
					return fmt.Errorf("error building raw array interface: %w", err)
				}
			case reflect.Struct:
				kv.valueLen = big.NewInt(int64(v.NumField()))
				if err := buildRawStructInterface(v2, kv); err != nil {
					return fmt.Errorf("error building raw struct interface: %w", err)
				}
			default:
				return fmt.Errorf("invalid value type %T", v2)
			}
		}
	}
	return nil
}

func buildRawStructInterface(m interface{}, kv *keyValue) error {
	mRefObjVal := reflect.Indirect(reflect.ValueOf(m))
	for i := 0; i < mRefObjVal.NumField(); i++ {
		v := mRefObjVal.Field(i)

		switch v2 := v.Interface().(type) {
		case []interface{}:
			if err := buildRawArrayInterface(v2, kv); err != nil {
				return fmt.Errorf("error building raw array interface: %w", err)
			}
		case string:
			// TODO: check to confirm it's an address (#1865)
			tba := crypto.PublicAddressToByteArray(common.Address(v2))
			kv.value = kv.value + fmt.Sprintf("%x", tba)
			kv.iVal = append(kv.iVal, tba)
		case common.Address:
			// TODO: check to confirm it's an address (#1865)
			tba := crypto.PublicAddressToByteArray(v2)
			kv.value = kv.value + fmt.Sprintf("%x", tba)
			kv.iVal = append(kv.iVal, tba)
		case int64:
			encVal, err := scale.Marshal(uint64(v2))
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(v2))
		case int:
			encVal, err := scale.Marshal(uint64(v2))
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(int64(v2)))
		case uint64:
			encVal, err := scale.Marshal(v2)
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(int64(v2)))
		case float64:
			encVal, err := scale.Marshal(uint64(v2))
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(int64(v2)))
		case bool:
			encVal, err := scale.Marshal(v2)
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, v2)
		default:
			switch v.Kind() {
			case reflect.Slice:

				list := []interface{}{}
				for i := 0; i < v.Len(); i++ {
					list = append(list, v.Index(i).Interface())
				}
				kv.valueLen = big.NewInt(int64(v.Len()))
				if err := buildRawArrayInterface(list, kv); err != nil {
					return fmt.Errorf("error building raw array interface: %w", err)
				}
			case reflect.Struct:
				kv.valueLen = big.NewInt(int64(v.NumField()))
				if err := buildRawStructInterface(v2, kv); err != nil {
					return fmt.Errorf("error building raw struct interface: %w", err)
				}
			default:
				return fmt.Errorf("invalid value type %T", v2)
			}
		}
	}
	return nil
}

func buildRawArrayInterface(a []interface{}, kv *keyValue) error {
	for _, v := range a {
		switch v2 := v.(type) {
		case int:
			encVal, err := scale.Marshal(uint64(v2))
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(int64(v2)))
		case string:
			// TODO: check to confirm it's an address (#1865)
			tba := crypto.PublicAddressToByteArray(common.Address(v2))
			kv.value = kv.value + fmt.Sprintf("%x", tba)
			kv.iVal = append(kv.iVal, tba)
		default:
			switch reflect.ValueOf(v2).Kind() {
			case reflect.Struct:
				if err := buildRawStructInterface(v2, kv); err != nil {
					return fmt.Errorf("error building raw struct interface: %w", err)
				}
			default:
				return fmt.Errorf("invalid value type %T", v2)
			}
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

	idxField := reflect.Indirect(val).Field(idx)
	switch t := idxField.Interface().(type) {
	case int, int64, uint64, uint32, *uint32, *scale.Uint128:
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

	default:
		switch idxField.Kind() {
		case reflect.Slice:
			sliceOfData := []interface{}{}

			for i := 0; i < idxField.Len(); i++ {
				sliceOfData = append(sliceOfData, idxField.Index(i).Interface())
			}

			for _, data := range sliceOfData {
				mRefObjVal := reflect.Indirect(reflect.ValueOf(data))
				for i := 0; i < mRefObjVal.NumField(); i++ {
					v := mRefObjVal.Field(i)
					var accAddr accountAddr
					switch v1 := v.Interface().(type) {
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
					default:
						return nil, fmt.Errorf("invalid value type %T", v1)

					}
				}
			}

			encode, err = scale.Marshal(encode)
			if err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("invalid value type %T", t)
		}
	}
	return encode, nil
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
	case reflect.DeepEqual([]string{"grandpa", "authorities"}, kv.key):
		kb := []byte(`:grandpa_authorities`)
		return common.BytesToHex(kb), nil
	case reflect.DeepEqual([]string{"system", "code"}, kv.key):
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

	storageVal, ok := val.Field(0).Interface().([]nextKey)
	if !ok {
		return nil
	}

	for _, strV := range storageVal {
		refValOfStrV := reflect.ValueOf(strV)
		for idx := 0; idx < refValOfStrV.NumField(); idx++ {
			v := refValOfStrV.Field(idx).Interface()
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

			case keyOwner:
				keyOwnerAlias := map[string]string{
					"Grandpa":            "gran",
					"Babe":               "babe",
					"ImOnline":           "imon",
					"ParaValidator":      "para",
					"ParaAssignment":     "asgn",
					"AuthorityDiscovery": "audi",
				}

				var storagePrefixKey []byte
				storagePrefixKey, err = common.Twox128Hash([]byte("KeyOwner"))
				if err != nil {
					return err
				}
				storagePrefixKey = append(moduleName, storagePrefixKey...)

				refValOfT := reflect.ValueOf(t)
				for idxT := 0; idxT < refValOfT.NumField(); idxT++ {
					key := refValOfT.Type().Field(idxT).Name
					v1 := refValOfT.Field(idxT).String()

					k, ok := keyOwnerAlias[key]
					if !ok {
						return fmt.Errorf("invalid storage keys")
					}

					addressKey, err := generateAddressHash(v1, k)
					if err != nil {
						return err
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
	case reflect.DeepEqual([]string{"grandpa", "authorities"}, kv.key):
		if kv.valueLen != nil {
			lenEnc, err := scale.Marshal(kv.valueLen)
			if err != nil {
				return "", err
			}
			// prepend 01 to grandpa_authorities values
			return fmt.Sprintf("0x01%x%v", lenEnc, kv.value), nil
		}
		return "", fmt.Errorf("error formatting value for grandpa authorities")
	case reflect.DeepEqual([]string{"system", "code"}, kv.key):
		return kv.value, nil
	case reflect.DeepEqual([]string{"sudo", "key"}, kv.key):
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
			err := addAuthoritiesValues("grandpa", crypto.Ed25519Type, v[1:], gen)
			if err != nil {
				return fmt.Errorf("error adding grandpa authorities values: %v", err)
			}
			addRawValue(key, v, gen)
		case BABEAuthoritiesKeyHex:
			// handle Babe Authorities
			err := addAuthoritiesValues("babe", crypto.Sr25519Type, v, gen)
			if err != nil {
				return fmt.Errorf("error adding babe authorities values: %v", err)
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
	if gen.Genesis.Runtime.System == nil {
		gen.Genesis.Runtime.System = new(System)
	}
	gen.Genesis.Runtime.System.Code = common.BytesToHex(value)
}

func addAuthoritiesValues(k1 string, kt crypto.KeyType, value []byte, gen *Genesis) error {
	var auths []types.AuthorityRaw
	err := scale.Unmarshal(value, &auths)
	if err != nil {
		return err
	}

	authAddrs, err := types.AuthoritiesRawToAuthorityAsAddress(auths, kt)
	if err != nil {
		return err
	}

	switch k1 {
	case "babe":
		if gen.Genesis.Runtime.Babe == nil {
			gen.Genesis.Runtime.Babe = new(babe)
		}
		gen.Genesis.Runtime.Babe.Authorities = authAddrs
	case "grandpa":
		if gen.Genesis.Runtime.Grandpa == nil {
			gen.Genesis.Runtime.Grandpa = new(grandpa)
		}
		gen.Genesis.Runtime.Grandpa.Authorities = authAddrs
	}
	return nil
}
