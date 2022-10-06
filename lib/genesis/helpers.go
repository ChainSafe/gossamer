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
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/lib/trie"
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

// NewTrieFromGenesis creates a new trie from the raw genesis data
func NewTrieFromGenesis(g *Genesis) (*trie.Trie, error) {
	t := trie.NewEmptyTrie()

	r := g.GenesisFields().Raw["top"]

	err := t.LoadFromMap(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create trie from genesis: %s", err)
	}

	return t, nil
}

// NewGenesisBlockFromTrie creates a genesis block from the provided trie
func NewGenesisBlockFromTrie(t *trie.Trie) (*types.Header, error) {

	// create state root from trie hash
	stateRoot, err := t.Hash()
	if err != nil {
		return nil, fmt.Errorf("failed to create state root from trie hash: %s", err)
	}

	// create genesis block header
	header, err := types.NewHeader(common.NewHash([]byte{0}), stateRoot, trie.EmptyHash, 0, types.NewDigest())
	if err != nil {
		return nil, fmt.Errorf("failed to create genesis block header: %s", err)
	}

	return header, nil
}

// trimGenesisAuthority iterates over authorities in genesis and keeps only `authCount` number of authorities.
func trimGenesisAuthority(g *Genesis, authCount int) {
	const (
		babeConst    = "Babe"
		grandpaConst = "Grandpa"
	)
	runtimeRefObjVal := reflect.Indirect(reflect.ValueOf(g.Genesis.Runtime))

	for i := 0; i < runtimeRefObjVal.NumField(); i++ {
		k := runtimeRefObjVal.Type().Field(i).Name
		var authorities []types.AuthorityAsAddress
		var newAuthorities []types.AuthorityAsAddress

		if k != babeConst && k != grandpaConst {
			continue
		}

		authorities = runtimeRefObjVal.Field(i).FieldByName("Authorities").Interface().([]types.AuthorityAsAddress)

		for _, authority := range authorities {
			if len(newAuthorities) >= authCount {
				break
			}
			newAuthorities = append(newAuthorities, authority)
		}

		if k == babeConst {
			g.Genesis.Runtime.Babe.Authorities = newAuthorities
		} else if k == grandpaConst {
			g.Genesis.Runtime.Grandpa.Authorities = newAuthorities
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

	grt := g.Genesis.Runtime
	res, err := buildRawMap(*grt)
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

func generatePalletKeyValue(k string, v interface{}, res map[string]string) (bool, error) {
	jsonBody, err := json.Marshal(v)
	if err != nil {
		fmt.Println("ERROR IN json.Marshal(v)")
		return false, err
	}

	var s interface{}
	switch k {
	case societyConst:
		s = &Society{}
	case stakingConst:
		s = &Staking{}
	case contractsConst:
		c := &Contracts{}
		if err = json.Unmarshal(jsonBody, c); err != nil {
			fmt.Println("ERROR IN contractsConst-unmarshall")
			return false, err
		}

		err = generateContractKeyValue(c, k, res)
		if err != nil {
			fmt.Println("ERROR IN contractsConst-generateContractKeyValue")
			return false, err
		}
		return true, nil
	case sessionConst:
		sc := &Session{}
		if err = json.Unmarshal(jsonBody, sc); err != nil {
			fmt.Println("ERROR IN sessionConst-unmarshall")
			return false, err
		}

		err = generateSessionKeyValue(sc, k, res)
		if err != nil {
			fmt.Println("ERROR IN sessionConst-generateSessionKeyValue")
			return false, err
		}
		return true, nil
	case instance1CollectiveConst:
		s = &Instance1Collective{}
	case instance2CollectiveConst:
		s = &Instance2Collective{}
	case instance1MembershipConst:
		s = &Instance1Membership{}
	case phragmenElectionConst:
		s = &PhragmenElection{}
	default:
		return false, nil
	}
	if err = json.Unmarshal(jsonBody, s); err != nil {
		fmt.Println("ERROR IN line-230, helpers.go")
		return false, err
	}
	err = generateKeyValue(s, k, res)

	if err != nil {
		fmt.Println("ERROR IN line-230, helpers.go")
		return false, err
	}
	return true, nil
}

func buildRawMap(m Runtime) (map[string]string, error) {
	res := make(map[string]string)
	mRefObjVal := reflect.ValueOf(m)

	for i := 0; i < mRefObjVal.NumField(); i++ {
		k := mRefObjVal.Type().Field(i).Name
		v := mRefObjVal.Field(i).Interface()
		kv := new(keyValue)
		kv.key = append(kv.key, k)

		ok, err := generatePalletKeyValue(k, v, res)
		if err != nil {
			return nil, err
		}

		if ok {
			continue
		}

		// if v != nil {
		// 	if err = buildRawMapInterface(v, kv); err != nil {
		// 		return nil, err
		// 	}
		// }
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

		if k == "Babe" {
			kv.valueLen = big.NewInt(int64(len(m.Babe.Authorities)))
		} else if k == "Grandpa" {
			kv.valueLen = big.NewInt(int64(len(m.Grandpa.Authorities)))
		}

		// fmt.Printf("%+v ==>\n", kv)
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

	// fmt.Printf("\n\n(buildRawMapInterface)  ===> m = %+v\n\n", mRefObjVal)
	for i := 0; i < mRefObjVal.NumField(); i++ {
		k := mRefObjVal.Type().Field(i).Name
		kv.key = append(kv.key, k)
		v := mRefObjVal.Field(i)
		fmt.Println("\n kv.key = ", kv.key)
		// fmt.Printf("\n(buildRawMapInterface) k = %s | V = %+v\n", k, v)

		switch v2 := v.Interface().(type) {
		case string:
			kv.value = v2
			fmt.Println("kv.value = ", kv.value)
		case uint64, int64, int:
			kv.value = fmt.Sprint(v2)
			fmt.Println("kv.value = ", kv.value)
		default:
			switch v.Kind() {
			case reflect.Slice:
				listOfStruct := []interface{}{}
				for i := 0; i < v.Len(); i++ {
					listOfStruct = append(listOfStruct, v.Index(i).Interface())
					// fmt.Printf("(buildRawMapInterface) listOfStruct[%v] =>  %+v \n", i, listOfStruct[i])
				}
				// fmt.Println("(buildRawMapInterface) listOfStruct => ", listOfStruct)
				kv.valueLen = big.NewInt(int64(v.Len()))
				if err := buildRawArrayInterface(listOfStruct, kv); err != nil {
					return err
				}
				fmt.Println("kv.value = ", kv.value)
			case reflect.Struct:
				kv.valueLen = big.NewInt(int64(v.NumField()))
				if err := buildRawStructInterface(v2, kv); err != nil {
					return err
				}
				fmt.Println("kv.value = ", kv.value)
				// for Debug...
				// default:
				// 	fmt.Println("*#* (buildRawMapInterface) unknown type ? = ", reflect.TypeOf(v2))
			}
		}

	}
	return nil
}

func buildRawStructInterface(m interface{}, kv *keyValue) error {
	mRefObjVal := reflect.Indirect(reflect.ValueOf(m))

	// fmt.Printf("\n\n(buildRawStructInterface)  ===> m = %+v\n\n", mRefObjVal)
	for i := 0; i < mRefObjVal.NumField(); i++ {
		//k := mRefObjVal.Type().Field(i).Name
		v := mRefObjVal.Field(i)
		// fmt.Println("\n kv.key = ", kv.key)
		// fmt.Printf("\n(buildRawMapInterface) k = %s | V = %+v\n", k, v)

		switch v2 := v.Interface().(type) {
		case []interface{}:
			err := buildRawArrayInterface(v2, kv)
			if err != nil {
				return err
			}
		case string:
			// TODO: check to confirm it's an address (#1865)
			fmt.Println("v2 at line-462 = ", v2)
			// var tba []byte
			// if v2 != "" {
			// 	tba = crypto.PublicAddressToByteArray(common.Address(v2))
			// }
			tba := crypto.PublicAddressToByteArray(common.Address(v2))
			kv.value = kv.value + fmt.Sprintf("%x", tba)
			kv.iVal = append(kv.iVal, tba)
		case common.Address:
			// TODO: check to confirm it's an address (#1865)
			tba := crypto.PublicAddressToByteArray(v2)
			kv.value = kv.value + fmt.Sprintf("%x", tba)
			kv.iVal = append(kv.iVal, tba)
		case big.Int:
			encVal, err := scale.Marshal(v2)
			//encVal, err := v2.MarshalJSON()
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, encVal)
			// case []int:
			// 	encVal, err := scale.Marshal(v2)
			// 	if err != nil {
			// 		return err
			// 	}
			// 	kv.value = kv.value + fmt.Sprintf("%x", encVal)
			// 	kv.iVal = append(kv.iVal, v2)
		case int64:
			encVal, err := scale.Marshal(uint64(v2))
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(int64(v2)))
		case int:
			encVal, err := scale.Marshal(uint64(v2))
			if err != nil {
				return err
			}
			kv.value = kv.value + fmt.Sprintf("%x", encVal)
			kv.iVal = append(kv.iVal, big.NewInt(int64(v2)))
		case uint64:
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
					return err
				}
			case reflect.Struct:
				// fmt.Printf("\n(buildRawArrayInterface) v=%+v\n", v2)
				kv.valueLen = big.NewInt(int64(v.NumField()))
				if err := buildRawStructInterface(v2, kv); err != nil {
					return err
				}
				// for Debug...
				// default:
				// 	fmt.Println("*#* (buildRawStructInterface) unknown type ? = ", reflect.TypeOf(v2))

			}
		}
	}
	return nil
}

func buildRawArrayInterface(a []interface{}, kv *keyValue) error {
	for _, v := range a {
		// fmt.Println("\n  =>  INSIDE buildRawArrayInterface")
		// fmt.Println("\n kv.key = ", kv.key)
		// fmt.Printf("(array element) v= %v", v)
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
				// fmt.Printf("\n(buildRawArrayInterface) v=%+v\n", v2)
				if err := buildRawStructInterface(v2, kv); err != nil {
					return err
				}
				// for Debug...
				// default:
				// 	fmt.Println("*#* (buildRawArrayInterface) unknown type ? = ", reflect.TypeOf(v2))
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
	//fmt.Printf("Value of i in generateStorageValue = %+v \n", i)
	val := reflect.ValueOf(i)
	var (
		encode []byte
		err    error
	)

	idxField := reflect.Indirect(val).Field(idx)
	switch t := idxField.Interface().(type) {
	case int, int64, uint64, uint32, *scale.Uint128:
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
		// fmt.Printf("\n missing Type in generateStorageValue = %+v | t = %+v", reflect.TypeOf(t), t)
		switch idxField.Kind() {
		case reflect.Slice:
			//fmt.Printf("\n=====?> in generateStorageValue = %+v | t = %+v\n", reflect.TypeOf(t), t)

			sliceOfT := []interface{}{}
			for i := 0; i < idxField.Len(); i++ {
				sliceOfT = append(sliceOfT, idxField.Index(i).Interface())
				// fmt.Printf("(buildRawMapInterface) listOfStruct[%v] =>  %+v \n", i, listOfStruct[i])
			}
			for _, data := range sliceOfT {
				// for print types of data.
				fmt.Printf("\n=====?> in Slice data = %+v | t = %+v\n", reflect.TypeOf(data), data)
			}
		default:
			fmt.Printf("\n missing Type in generateStorageValue = %+v | t = %+v\n", reflect.TypeOf(t), t)
			return nil, fmt.Errorf("invalid value type")
		}
		//fmt.Printf("\n missing Type in generateStorageValue = %+v | t = %+v\n", reflect.TypeOf(t), t)
	}
	return encode, nil
}

func generateContractKeyValue(c *Contracts, prefixKey string, res map[string]string) error {
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
			fmt.Println("\nERROR IN generateStorageKey")
			return err
		}

		value, err := generateStorageValue(s, i)
		if err != nil {
			fmt.Println("\nERROR IN generateStorageValue")
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

func generateSessionKeyValue(s *Session, prefixKey string, res map[string]string) error {
	val := reflect.ValueOf(s).Elem()
	moduleName, err := common.Twox128Hash([]byte(prefixKey))
	if err != nil {
		return err
	}

	storageVal, ok := val.Field(0).Interface().([]NextKeys)
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
			case KeyOwner:
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

					var addressKey []byte
					switch key {
					case "Grandpa":
						addressKey, err = generateAddressHash(v1, "gran")
						if err != nil {
							return err
						}
					case "Babe":
						addressKey, err = generateAddressHash(v1, "babe")
						if err != nil {
							return err
						}
					case "ImOnline":
						addressKey, err = generateAddressHash(v1, "imon")
						if err != nil {
							return err
						}
					case "AuthorityDiscovery":
						addressKey, err = generateAddressHash(v1, "audi")
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
	case reflect.DeepEqual([]string{"System", "Code"}, kv.key):
		return kv.value, nil
	case reflect.DeepEqual([]string{"Sudo", "Key"}, kv.key):
		fmt.Println("common.Address(kv.value) ==> ", kv.value)
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
			bKey := runtime.SystemAccountPrefix()

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
		case "0x3a6772616e6470615f617574686f726974696573":
			// handle :grandpa_authorities
			//  slice value since it was encoded starting with 0x01
			err := addAuthoritiesValues("grandpa", crypto.Ed25519Type, v[1:], gen)
			if err != nil {
				return err
			}
			addRawValue(key, v, gen)
		case fmt.Sprintf("0x%x", runtime.BABEAuthoritiesKey()):
			// handle Babe Authorities
			err := addAuthoritiesValues("babe", crypto.Sr25519Type, v, gen)
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

	if k1 == "Babe" {
		gen.Genesis.Runtime.Babe.Authorities = authAddrs
	} else if k1 == "Grandpa" {
		gen.Genesis.Runtime.Grandpa.Authorities = authAddrs
	}
	return nil
}
