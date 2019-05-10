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

package trie

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChainSafe/gossamer/polkadb"
)

type commonPrefixTest struct {
	a, b   []byte
	output int
}

var commonPrefixTests = []commonPrefixTest{
	{a: []byte{}, b: []byte{}, output: 0},
	{a: []byte{0x00}, b: []byte{}, output: 0},
	{a: []byte{0x00}, b: []byte{0x00}, output: 1},
	{a: []byte{0x00}, b: []byte{0x00, 0x01}, output: 1},
	{a: []byte{0x01}, b: []byte{0x00, 0x01, 0x02}, output: 0},
	{a: []byte{0x00, 0x01, 0x02, 0x00}, b: []byte{0x00, 0x01, 0x02}, output: 3},
	{a: []byte{0x00, 0x01, 0x02, 0x00, 0xff}, b: []byte{0x00, 0x01, 0x02, 0x00}, output: 4},
	{a: []byte{0x00, 0x01, 0x02, 0x00, 0xff}, b: []byte{0x00, 0x01, 0x02, 0x00, 0xff, 0x00}, output: 5},
}

func writeToTestFile(s string) error {
	fp, err := filepath.Abs("./test_data")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fp, []byte(s), 0644)
	if err != nil {
		return err
	}

	return nil
}

func TestCommonPrefix(t *testing.T) {
	for _, test := range commonPrefixTests {
		output := lenCommonPrefix(test.a, test.b)
		if output != test.output {
			t.Errorf("Fail: got %d expected %d", output, test.output)
		}
	}
}

func newEmpty() *Trie {
	db := &Database{
		db: polkadb.NewMemDatabase(),
	}
	t := NewEmptyTrie(db)
	return t
}

func TestNewEmptyTrie(t *testing.T) {
	trie := newEmpty()
	if trie == nil {
		t.Error("did not initialize trie")
	}
}

func TestNewTrie(t *testing.T) {
	db := &Database{
		db: polkadb.NewMemDatabase(),
	}
	trie := NewTrie(db, &leaf{key: []byte{0}, value: []byte{17}})
	if trie == nil {
		t.Error("did not initialize trie")
	}
}

type randTest struct {
	key   []byte
	value []byte
}

func generateRandTest(size int) []randTest {
	rt := make([]randTest, size)
	r := *rand.New(rand.NewSource(rand.Int63()))
	for i := range rt {
		rt[i] = randTest{}
		buf := make([]byte, r.Intn(379)+1)
		r.Read(buf)
		if !keyExists(rt, buf) {
			rt[i].key = buf

			buf = make([]byte, r.Intn(128))
			r.Read(buf)
			rt[i].value = buf
		}
	}
	return rt
}

func keyExists(rt []randTest, key []byte) bool {
	for _, test := range rt {
		if bytes.Equal(test.key, key) {
			return true
		} else {
			return false
		}
	}

	return false
}

func TestBranch(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("spaghetti")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("gnocchi")
	key3 := []byte{0x07}
	value3 := []byte("ramen")
	key4 := []byte{0xf2}
	value4 := []byte("pho")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err := trie.Get([]byte("noot"))
	if err != nil {
		t.Errorf("Fail to get key %x: %s", "noot", err.Error())
	} else if !bytes.Equal(val, nil) {
		t.Errorf("Fail to get key %x with nil value: got %x", "noot", val)
	}

	val, err = trie.Get([]byte{0})
	if err != nil {
		t.Errorf("Fail to get key %x: %s", []byte{0}, err.Error())
	} else if !bytes.Equal(val, nil) {
		t.Errorf("Fail to get key %x with nil value: got %x", []byte{0}, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}
}

func TestBranchMore(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x01}
	value1 := []byte("spaghetti")
	key2 := []byte{0x02}
	value2 := []byte("gnocchi")
	key3 := []byte{0xf7}
	value3 := []byte("ramen")
	key4 := []byte{0x43}
	value4 := []byte("pho")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err := trie.Get([]byte{0})
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, nil) {
		t.Errorf("Fail to get key %x with nil value: got %x", "noot", val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}
}

func TestPutAndGetOddKeyLengths(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x43, 0xc1}
	value1 := []byte("noot")
	key2 := []byte{0x49, 0x29}
	value2 := []byte("nootagain")
	key3 := []byte{0x43, 0x0c}
	value3 := []byte("odd")
	key4 := []byte{0x4f, 0x4d}
	value4 := []byte("stuff")
	key5 := []byte{0xf4, 0xbc}
	value5 := []byte("spaghetti")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	val, err := trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key5, err.Error())
	} else if !bytes.Equal(val, value5) {
		t.Errorf("Fail to get key %x with value %x: got %x", key5, value5, val)
	}
}

func TestPutAndGet(t *testing.T) {
	for i := 0; i < 20; i++ {
		trie := newEmpty()
		rt := generateRandTest(1000)
		for _, test := range rt {
			err := trie.Put(test.key, test.value)
			if err != nil {
				t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
			}

			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}
		}

		for _, test := range rt {
			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
				//trie.Print()

				tests := ""
				for _, othertest := range rt {
					tests = fmt.Sprintf("%s\n%s\n%s", tests, othertest.key, othertest.value)
				}

				err := writeToTestFile(tests)
				if err != nil {
					t.Error(err)
				}
			}
		}
	}
}

func TestFailingTests(t *testing.T) {
	fp, err := filepath.Abs("./test_data")
	if err != nil {
		t.Error(err)
	}

	data, err := ioutil.ReadFile(fp)
	if err != nil {
		t.Error(err)
	}

	slicedData := strings.Split(string(data), "\n")
	tests := []randTest{}
	for i := 1; i < len(slicedData); i+=2 {
		//t.Logf("%x\n", []byte(slicedData[i]))
		test := randTest{key: []byte(slicedData[i]), value: []byte(slicedData[i+1])}
		tests = append(tests, test)
	}

	trie := newEmpty()

	hasFailed := false
	passedFailingTest := false
	rt := tests
	for i, test := range rt {
		//t.Log(i)
		if len(test.key) != 0 {
			err := trie.Put(test.key, test.value)
			if err != nil {
				t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
			}

			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
			}

			failingKey := hexDecode("26")
			failingVal := hexDecode("dddd1d7afafbac50b56baf7182e1e0bd3cd99522c239cbf3a475a134af")

			if bytes.Equal(test.key, failingKey) {
				passedFailingTest = true
			}

			val, err = trie.Get(failingKey)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", failingKey, err.Error())
			} else if !bytes.Equal(val, failingVal) && !hasFailed && passedFailingTest {
				t.Errorf("Fail to get key %x with value %x: got %x", failingKey, failingVal, val)
				t.Logf("test failed at insertion of key %x index %d", test.key, i)
				hasFailed = true
			}
		}
	}

	for _, test := range rt {
		if len(test.key) != 0 {
			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Fail to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(val, test.value) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
				//trie.Print()

				// tests := ""
				// for _, othertest := range rt {
				// 	tests = fmt.Sprintf("%s\n%s\n%s", tests, othertest.key, othertest.value)
				// }

				// err := writeToTestFile(tests)
				// if err != nil {
				// 	t.Error(err)
				// }
			}
		}
	}	
}

func hexDecode(in string) []byte {
	out, _ := hex.DecodeString(in)
	return out
}


func TestUpdateLeaf(t *testing.T) {
	trie := newEmpty()

	// case 1: leaf -> branch w/ two children
	rt := []randTest{
		{[]byte{0xfa}, []byte("odd")},
		{[]byte{0xfb, 0x0c}, []byte("noot")},
		{[]byte{0x0f}, []byte("nootagain")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	//trie.Print()
	trie = newEmpty()

	// case 2: leaf -> branch w/ prev leaf as value, new leaf as child
	rt = []randTest{
		{[]byte{0x0f}, []byte("nootagain")},
		{[]byte{0xfa}, []byte("odd")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	//trie.Print()
	trie = newEmpty()

	// case 3: leaf -> branch w/ new leaf as value, prev leaf as child
	rt = []randTest{
		{[]byte{0xfb}, []byte("noot")},
		{[]byte{0x0f}, []byte("nootagain")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	//trie.Print()
	trie = newEmpty()

	// case 4: replace leaf
	rt = []randTest{
		{[]byte{0xfa}, []byte("odd")},
		{[]byte{0xfb}, []byte("noot")},
		{[]byte{0xfa}, []byte("nootagain")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	//trie.Print()
}

func TestPutAndGetBranchKeys(t *testing.T) {
	trie := newEmpty()

	rt := []randTest{
		{hexDecode("bb71f24c5760cc54b8aaf5aacc1d27af56a62fe209dbeafe8f9d09f37f09176f5225803929256bdb011e1bff44577b8d4d554b21afc33b31108ba213545ec547ddda6db87f5db3c747f1de63944260c673b2b7d6bb706da312cde73a37266d68f1f1cc87576ded2f3143947b5c4885fe18fa9b095114bc7764377918d5cc8eb0fe63005b9192907ec7e26b405991f5ab920423a329f4b022ca6ed39eff103882a37cf9de54a8f9ca07e47630262be61d306c0c68a036ee5100f243e754cda180529225e02120700590c569c1e0a0809611b45bec06bb2fcef09b4657862ad5"), []byte("odd")},
		{hexDecode("00c0392cecc86564b9ab0278c03d9f716008762a1afe763924fc8dc2db50237716124e17658b06710863c1a1f0c3fc28b629b6dbdee89576f1b7adab701540baf7d6fda1947c818bc8f1791913e2c95b281dc2a47e3eec0743d88c6402b05fb53d7b65ef7db9e2441d37579c5ee3915e8720ebea3341c4110ce74d62bea996ab2eaafbc87204cf5d2963cceff1241b6a489ce3"), []byte("noot")},
		{hexDecode("02033f19a35cb316ff0a9878d91397e2093954333856b16f377e610bc2f89788b63ba10037dca291e1e56eeffa23f0a25b86d8bab5ef4178b0afb190388d58b2388b35a1bf9d96a3c8622a0b5885bae2237e19715098775e42e9a8c38f555b2910ce876c155aea8725cb0c84fcd0d8911283cfad383e7ad86120303970a8"), []byte("nootagain")},
		{hexDecode("12d78c462451d1f634c7d4df102961f28798c5160279b57d82282e3d6a27686eb8551192c96f89e7ca681ad54b22567256a52f85456d239d1ac8e2456df6ebe9e460cf57cd8db9314da044b17a645ac8d3013f62b41cdb52fc8d6fdee25d73028048a1a885a3a147aedd7d5a0e5599d8a8759f06778c6076de682af1d16b65c1ab5896d3e5c81c029dddea401c67e34789967ddb5dc2869ed5f6bc2e052030be8fcb13e0aefa5253cf670a8d4582a6ae865b7556b3d06dcf7b7518e391bc5c2f98f76299ef02b644bd57222e87580b983a9c90446331e4f903b07e33ae461975c409d6f9aff3dc147c4ca0031364adb2051d7e2bba953272c8425871dafd9ea564ad07bf468d39dbaf544d3866abed42a572422889621a02ed729dcd4317861503f18b96f7506c9636fbc8d7f4ad76804d200692861bb9775abbbdbfa77f51b7cf04adf9377f36953cb30eb02380effac1e600c1cf4c9cbcbe"), []byte("odd")},
		{hexDecode("1a1f43bd0ff8bb70ed5a4764ace5ff827035fa2ef2a627f24fb254ed3d555837cd254f537ac83ba14f7167757c233a6c0f50fe7e1299326d0d3f3d02a81cff64a82617f77e858753791f8f86dfef57dab2550589a596b57d6c51ce7acc8a1a8d8628e4fa3cf205c0dd5d04a9e7a44b31357d1e4ad8571f4950020354c6be9f9323d4f6ccc6f73707ac1fc25b26820280817600"), []byte("stuff")},
		{hexDecode("245d49556eb987b4f72d9ad22fdd2ed6e29c6aecf88173fb8d5b2089a0fdc64687f4245cb7b9e28b256255d760150998f04e61ed376ed2c569469326c11f12a40b4f93550ebdc3aca20e05a13d461eec59f998b4a3026d5ecfa3413028ccc1be4c52f444f1cd6394425f9972f3b9e08b613ae7434d3055650a11c936a426c51d7cf5b081b277b4b431854e3255f05329981f7cf2d6e140"), []byte("odd")},
		{hexDecode("27739c84fdf6236d850ef236ff0759df7732ef37dc7411814a41635bf928d2afbf3f6fe47be7644073f7f2587da2dad2e39cbe703872368ef349eda7aa0a682c9e1d14002d4bb014eed91ee465b845214e7ce833180e409122f1cc20d795b1cf736fbf4fad48eaf8f846100b8ed140b914ac4fbc76f0a825094fd3cc3b74150c3ef6d81838606deff1c2d06a7e47ae7b11648fa3abbc76d7c33818ca1091913e8e7b420ddf0eeea4ecbe085110e3a2f49eccdac4e3115ae2ea896ad4466e575bbba2156a85434e9c57e60e6ea334ca3195c7323330dfb15572eb080a9fe09d419755e3816acab83c9a93d3c0d197dc5353e20ec7cbd3b0eb8af459aaa01256c47e172d5f7b2e007b7d05dce26c7d5b69d812b0bb0ab7427072c238d4ce18595a049a96b58d3185310e038985ec39daa0727ca8452d2ca609e6103c"), []byte("odd")},
		{hexDecode("2f809a2ce58ee9e3ba57f352902814cce0fb1a7b7566d51ad20882f2fea385ce9ac823ba6093123d4bdce2ecfd6899831ff48ab60750b61ffba5e8e0216c393229ae412caeceb315fd8e9bd95761aa5ef66f409010ae09339155fa32b7cd868f663ae43dbf90d80676a3b6b2648bae32aec560b97e820f8aa23256af55a0ee8e83238def64ee16954fdcb085ffcdf20d92a8be8d659ebbe999d63636b7d8410cb3c8c8f8bbc0831c989fb442dfc699145a60b955f4909c57ffda967f1c7a4cbe34c57546dd8d9b0d39463ab24f038e839a1f52bce04136bf4d9756f4e4fdfc3f46aeec7c30d2f9cc62f75c4226fa738c6f10daa97bdc7e8805022a0923d7f88365f9def1099ae4fd5dc0f121d3986cc4a65e4185594c4d9f9814652e5c5792d1ca8a47b57160dd03892248c6cf3510144aa7f85015f7b1202d5aa9665897256b460f6675ca8b65be98824bb0028e80c39b8c35db"), []byte("odd")},
		{hexDecode("3e5142d079ffff0f87b97f4c2b8a30357f497d72e7fbafcb70d2f0e8e47548c8d00690bd5d4390841a26b03d5a6432394dd8061c03c1d72a70dc6202b1cc621ce2aba0deb7ecd4a9626ce75ce9bf77cd237c8031406f94c570958afb2c3e36ba56d1ad13a7bebd716b11d52f378b5f84adb2bb1886d7e8ca58a4"), []byte("odd")},
		{hexDecode("3f6ebd812f5598d2aa91e29570219df168ae24e2a218ecd8b7b936441e00cd988346d3df9fecf22c639eded4b00ee09f0c0e9f0f1ae2b9a61485c66dfefbb9d754994e905ae6b100d0e1366fe91f4184032a556a041601b96209d3720b599e5453a50cfb3db68f7c5c5b"), []byte("odd")},
		{hexDecode("41d709351f4246847bc2320b58fb8c11131b8d2ce10cc6d3ef84424157528dc5bfc5a5656628662c9171b93df5fabac940ed1a84132c53b2a31d43f943e76e325db21b45afa518a8d608bb17ea0e7aabc9f7d2085dd633368632bf74735c4487445f01cccafb465cdc8e58e08d37d373bccad1e097220a"), []byte("odd")},
		{hexDecode("456c9c71fd4e0201bfac95fc54db6f2a207f998d4e25681bb3666b53962c454f4b37b1a4438c44be993fb8112c08e39433a76e4060628fa2dfa1c7b9397881166d903500197932dbba296f8c7c18d58cf4ca17ec6d5da334654a5ef2329e6e74447b01e50c54a9f1dd756001f612ff85e8aae25b54311f8b2182ed76834cc2a8f71819805385f8278ad681ecc32908a0393d458a91dcb46eb1fba29ad1c7d86e8bfe6743ab7f4b1249929f1aaa7fe404c2d2b8996b9f50abeee019345ff87aac3e02dae7dfc2cdc83222b07a302e9c84a4795c1b39a1ad8b3e9d5f79419eecc27eec4a25772839484f3a8f6a199753a1922fc9188a11d07b61c49605e72fdddc9b998ea81fde561f83e5917fa051946b3a13c9c9cfe06efc4418082dec3ddc2db9a1edf30a7ffb43c781cb33801a5e85ec8320f6ac2382af72d24495bd479c18b9b748b2f74a13482b446bb7b14cd7a2d38d3913835ee481bb913e66a198fdf428177f4c2e3681089d28cad9c1d46bdc23e00a00da"), []byte("odd")},
		{hexDecode("4620c43940919df36af8e69603479a16cc6e145334b426a3603373b3382d414cdd448c8098f3dded28efb2cd0aa78b5a9492d2c3f478f36d2dff9cc8c90e8c81fe6370cae66b3ab0d703cff463455a5f0e1934b75a59992207d4c2a162fedf7fdd307becfd24c0536be19c4401f9ac607949d36e3a350929b5e7c5acc90ab7f8a7c90512f026f1361f2d97d868e081c08be2892259e453ab28f926bd8dca03a6212e3c42df2881262785213db3886d2a1d50122eac00caee86eda1a4e3b0d1caec7f6610468f1bebd973ace6c0df592d08846f7a779e1f9d26e1c670fb6de2ac863c8cbaa83d77bf8384bbcf58a695956c9590ca60723a9dc60b74fec46218fd4fefbc67477c7d830b81bad4f4610f67e3277d938b9738ae2fdbc19f1c4812ccfbfb7d2c122f9ba51524ee3845f5b91d9ae4656424a281cdd5271abe60ef6288d02963eefb1ec15002c5674b1c107a972502a8599763ea6ab03a082cdb955e1d61c7565dad2e800c2e77f8c5cc7c5392f2ae7abe96e1d2cca9f6"), []byte("odd")},
		{hexDecode("4760ffd13f6612d522e8f05dcd9138e556ccb7f67d88e62792ca50ac908cbb"), []byte("odd")},
		{hexDecode("4c8a4f80b2a4adda8d647a0d69d1dc25b0cf15b9ad17a35b448f54c471a94f81ad6dda1c99da6074339d4a70b12887de25caf9bffeb0b160c413fbf27fc6445172c564057837e55fdf182672a485adb1623f5e503c709816934fa0e0ddd76c5fff2b445275f338ac52cf8465ef0f29d6138817dd2b23132d794a2aa684b2d60185135e39f8f16bf622d194d326bee0fa8e1f176c3ce182461c637f0cfc38c2fb7c74f7741c74be3dd521a630d1d29f82a22259278033401e71e8eae842912e5e89d694ac81e1fe82ec0d10a6dd8bb6c5b8dd21569b862ca6979600583de02383675a9517b7ce2918123dc326b469819ecb4b46bb9d80cb361a98b7ea80ac039bb84ce194ef2132793f20ee2cc3f30af2bb17ad6c3ed7c7c9"), []byte("odd")},
		{hexDecode("5b99998fb2521804e09c281673cb3112fd85bcb78153b83565c402db17979b616f20e8f57af3107343f0e30b8e1223c4f57cd43567f5c9f1b84767c107e6cb5fc823b7c70873b286a73f726906c3c907482e59ee2a48a40708ce24c2753648e44141522c829551a333a1cc1a8a13c255a540e6c6763ea0f20487cd36512008442577760385b3e04200c00a5888d05529f747a73152bad3b8a418b4ce67a77d7d622173360da6ae542e4b222d02ea83d4936f36a1c4fcaa6470a4577fa8abbf339e982114b39969f71c7f18d48dba2bb93edf9ebbc96e9fcfc76daa93b500cab96602812191a72007fd43ea4be3613699b1f77e6aa46d2b1918d27076760df6b321b616da356e44910d615ac3d3af3c396b2f4708bf99ca8807d794435a062e23273409e7e4ccfd4151af2ed0"), []byte("odd")},
		{hexDecode("61d31df37757eb825b251547edea36aed88038de70e79f2aa9d115b15e5983200a101a90dcbd04369707c373797270a26da7fb657da67f36a97fae640ed34b36ac4193bc30b23f522c5df134c375eac82dde9295356011a115b3d23be4d3e4b05866283bd06e931646de12d7d13bd98db0ef8aa3180eca534e6e0709c062221f8d574e004dab2bcc48be1fef112367ecb8d9ef4bc48a5937190fc9335cfe4ab974a61983ef1a57096037da5e7f05f5f9a70cfe252def785b04b3be3853ea90d99c5d9bdebda61a9f27403e56c75947611b60166f6195d6828eff05c3a92c6208d5ea99704ef8fadcb6b7d2b5c1cd8b7d46cd35b87d0c6ed8f2fbfad533346c306c652de1306808ccd665a044f39573b3e3fcc30484e6c027b6ede27c2faa3c901e24fc250e44f92ccbf332257e9c41f4a5dc04"), []byte("odd")},
		{hexDecode("624453a0e2a4624823fd110b0feba99db82e757cd3ebbfc3a52605f5bcbac70df5ed8cdc1118860d20348a231c2b401e614975fcfc4fcba7fb4fc2aa672fbb2154260b3d7973e0776636b5e420ee073a8bee0bf0ca1be478aff370496833dc3a5ff46945878d535d2dfbfd0c6e72b87324b4023842f63d9aed5f5b3854dad0ac8ae30db64183f84410c11831d0"), []byte("odd")},
		{hexDecode("65429eeca032d41548b5172c1e55decf4ce246"), []byte("odd")},
		{hexDecode("659e5d86d7192b7749f320586654cb2ac2bd9aaad1819edd8184cfc003b1ffaaf3560f023c285723499a28b6c28fa6ec2bad96a1dc250a9c24aec5a8427910eaae8ee6e5bb725555160255d2ea529f8e6166e0895135dc77a016a22abd90fc05fb483a90a0d936ea863280c0dec610dd1ffe7299aef0e4acdefa04b5481d9625d161585e33750fa0231cb6eeb35cfe81adae928c9e30a8621477ee28f0797818ddc8a9ab46e0c1d5a57030b6bf01251647b7daf1dfab711457c8e364f4db356808fe9c04c555836557528d73e8907d7cef8671d1f3351ef2733585306b70f43755e3d7a7563765f7a45b54aafbb4ac16a1a9bfa1c2b41ce7ad2f4d9ad8902a470ca3a021ff59a68e686e325c02fcf44f294d137e7f66eecb"), []byte("odd")},
		{hexDecode("75c67ca30e10d10e394ed56843bb9fb72af841eaa1b585b8ca86d4a3d9310f592932b7b85123a0c2613f2caa28a2cf1fb8312ccf95730d1c707a67ba3c90f241f2673a52bd233f475f3d6e77e820b7732638c9564e843f4df014d16b4289afc59b284c3896ac0ebf63d87c67e7bb0139141804b71fcd7da53ad00927cbfafbbb83c125e4bfdf3bb42e91662635da6af1928fefc7829cbfc99f6327b5e714fad76652f831f38c1b1a36513c593b52513960f74ce1b4ba7c2c9a225005809fe17f10a1b37851baa99cc2163ade7bd7295680e3799a9ef00eae948ede852c1bc99f4a084dd9a8bd7105134c113afb8e17e4ece9652fe9d1a5ae324c5962a9a567b56302552e2e148601b3d779ca4578a7aeec2621fbd574fcc89dd1a4f25e785f76e7b2a7bb2f6cdb8a9deca03440fe25cdfa16cb4081"), []byte("odd")},
		{hexDecode("8387a742f0c3133e941b5fe0e57dd77accc901275c21a84f4e4f866bac0de0d1eda873418ed5f9f340353ad76c912e3a115f54f291c6d9ddf29aecbac7c3d318b11dff6635bb63f892dd9789912b5e14bc9362dc2ab12ccbf23f100a73f3570122d85cffac166564fc2065a36d3372ee2cc36e26a41582b836e5a8df747320abd69492a02853bba5d7e0bec5895368644dbd0bad8df1c1ae27d89a2ca713a4e279ea58241b089b5d8641706f3807831770f06aeb4428f736d50a353cd746c9729c"), []byte("odd")},
		{hexDecode("8b00878ebcf9b05b4b6ed9bdd4e97e94b817fc6482e5a5878187abaf426130106c9cf3394f01f0aa2f5f5d452295e1a54a9c024fc62830465bb0d1affc8775bf418c07ebcb90898f4db700ee0da596d8199a8a600468e2a196ba95172cf1a47799a01813910d1a64e4b6457bb1402c2140962f5a56859fe835768ac0abeb6676a2e62864a327e10f852885770a0cd5e74754b7c566ea731943acbbdc96df18727482dabb9da4c4315882028aa5cf63b514791b5576fde02fac657ac03d6e3c49d3d37cb4990aecc981a46757ffcbd660a1b45e5426e7849b200ffb51c50f6dfa38865054cd80691aadb653af6a11cadb41e39b904413c52347869272e321566f36301b38742236d9849d84cff4115af87ad69a2b590e"), []byte("odd")},
		{hexDecode("8db390c64efed96c02fdc82c178dcd389df1f9bc274967b93d08b4b77c1d543ebc9438c8f53bc8c171ce7660d405c0f0c24578a475351d18e3a1b926dd536509d29695c3585d5762990496f65047411e180990d5c1ee8ec29b0613aa26d6ff15e282b15968e480ff731b8dcd3e52eb615a6f66978141c1b093cb893086a48ba538dc98dd64081893605652241a662b9993a2a378bcd9d4573797c77279a3bdf11542ecb08d13b10acb7926a3e997cc62cc6c9831838f5e28c38c2c3760f201ceea3d0d28c03910ea4e009afa4c603458fb87fbc95840233b245af2f1994c705947084e246dd00eee05d3178c5599bbc5f99bf3dd654302cd17bb58e532bdfe113159d1eba4a3c6ad04a1eb265da98fc0fe8ee63980473d780b49bcb17dcfe30eb17c561b3d892d076e8dcbc660798a3df5ec11a3ca11d2d01919982f530a75b080058e61b29b39a1f1f5d7b178f2a3"), []byte("odd")},
		{hexDecode("8e07a55fa5590201298845edc1b4b563e1761f73101eebe2ed4777edede7f0dcec28f33f09b6b84015500d3f712d45f0b99b909d15ae5a5f1312e2e3ee513d677f8eb55d55873b589947e97ebae02dcb7385f5a03219c4c846a9f5e1303254235b6f7c88c14b558eece3e39942572b9eb0a4c4bf08e5906f9f1eef87755d1085295392bfc92d24c9a891ef886618c70437761fac9e25331814248a93f2598e26df766c28bb7ba165d7fd869b6c77602aebbf449caaaf7320898da1c988dc5cecf80a60c34b478faa0d40abe0c70de36adf6d270429e2a24cc0"), []byte("odd")},
		{hexDecode("96c6fba2e5cadf2d6b663a4388f10bb290c5e25d0e372c1cbc3305554a39533e666be3ae10b73b1fe569b6ebcfa1c13a0c8b1243f0b3da8ef0316b519ce37bccef4d0d6b5970aa7be463835f34b3fccfce36682ec94c8e7abd3876f39ddd5a00f3d738de9890fd001193064f3ce9e4da1d66a2ccfd583748538edd1d133803d05b2a68393c9ab548e7573c255f5e2c082f262e3ba3c422ec132d2e9541ee63899e162ccf2b0ccd3434de0eb1312cd2fd5810dedd1c3631640f143b030688f24fba8a90ae5b97be184e3c1e9c35d1a15ee1e019ee1cae91701dcf42e25c47d689154c63b9fe693905"), []byte("odd")},
		{hexDecode("99a52bff5a9827b6cee22da79f59ebb4c662e00c74e4614bc35687dd04a6c432afb6976f11d1263b757a81dc2572d5fa7b0df1085b8e005a1b5049a5b6be714b307e1580ca7649f05531a2e2b56d2d59493231c90b82e9b8f6aaf62788c9709bf3c34a3dd7354e16d3cd0a337f0481e694a8131e4820fdba6a4deac0d1a305d4993da87aeada4564a85edb7e5f76e61c0eaddba9ebc2862b3f62851f1b18b21b3b404cddea7e40e97907f26ad6f21649bbd96ad4852135c26c4bbf9af3addb71ac84c6bb657401ffb4d3534f0610fb99c51b761d5227ceb3c1f49d716cc9fc56d8387150c41e74a9daf1a4f3a1870a26a1ae0193e6a1c6028a1ff384b7a8f6ea"), []byte("odd")},
		{hexDecode("9b5763717ed16979d350177c7e0c1d9ff86e0d0a70b7834c686cfb0682fe621e1083d34a77128228b64df146f5ff6a9b05b3bb83eb9ad1da52200389d3b96769956be6fa94aacae955c8a862cb35581558d1f8b5bd4fa09209ca69bcdb0dce41bf2abba13a1c67b99195e9fdc6751b103d120b2bd3cadf2bc3f49adc37e44aa0a8e15fb451a53d86bca1ee7e4cab3664c50f4ce3fd01a041b65077b946ed5eb95b653124e16411907344871b244e1c74932739a675ad71a53ad1c870c7e3c8013ef8e19e8316d836accb3fa6e07a5f57bcb0e358829f609d66f1616ab9cdee5584ef107c1d432dbc5b4832b92fbe439d554c6dfd97e14329258a6b199fe66a38028c4746ae0872652210330d4923d0282c5f346d09c5a2f0ae4c968a16e94b8137a541b23bc57f9b7581d9729606c2c43c6077a7316fff8e8b0e0e085486f3e494fb34c092"), []byte("odd")},
		{hexDecode("ae7a6593c34402e0f3bd40cddee8a7bb3feea64a2f14b3bafe5621161d4df7ea3377947d18fe8f65b90a74cb78e3ee92c90498abfe0bc2295db6cfcf019c95d876293df1a139ce59ba09a70fa558666d76d925236fbf3f336eb9990d3164f08995c0d0012bf49da572241e8fa333f3fd2bb7b3de226d21708976e82fbb65250cc440dce1e95f49e75c4467eee6e4151e2d3c8d73fd782e214081e32ac8b293935ad4e0e89f6cb9b547050cb35c4e787ae48cba9e4e54af3f498e63eb7d19ae14e2762ccd418e5bc017eff55fa074d600c7804f41b3f155cb9bf2af00fe5af69e452ca516925e495a27dd3f847d7318baef528e04fbc578994a20686d3f92de5e3afc004fa52e91ddb2a40094301ae9f5ae016395ca85e5803a9d97"), []byte("odd")},
		{hexDecode("ce6ae94479ca9ccfc52bb049f7029be0463575e84176da5530d9e8b27e818610ffb1e1557ef617be532db3dd52f987cfba06eade71f594effb5664becd48c51762979494b087a01e3b575125b3af79481b40e805c16260539c03ef99548e0f362b662c18875bbf812bff278668a4b43c25afab19f5396a4fb061b3f79876d54a2ec50fa3d72543364c799c95f4c50ee605f5ef1debf4d43a58f14332ee4f97161eee0145605b3d2c4282d086ec400ea8f5bac835f626aab9a530dbacd88065802c29557d22cefb5abb0fda7391cb176a3bd9cce1a8c366cd17fe03d6c0a3adf1632da966094ebee8de5d474257ed8195bc4def2a5cdae71ccff5d0464dbd3c89595b70405d1a5d3cd21cd17635991848b6a5021c25deb92de85bbfbaede592f2a0b95da861c76485fcfc97253580e039676e6e067b3b39fa4ef2455164935a5b0a156afa57385347c56cff5c19c9312de217"), []byte("odd")},
		{hexDecode("d575a0fb65f39e1f46c9f1e8394764754f19c2a43ba8a01ce498ac5669e149d4f94163ff94c36897bf76e69580c09198c0edce411c73b535f2f0a6c80ad9fac8122c0afe3fd5ba7c80f3311171d764ca15ef75640a24a762d18f532ec34766943805094904b83ae7e2f4f994b2a791f787cd94429327fa393b1475c3a29a1c84066c6b1c540ef92d680e039fe88be24273c197721e6245966c6bb85c2a1d500ecd0c9a9676ee9d5eaef9baf6f399bc52de5faf11e060e3dcfddb0b9cca2c5a7133f7d2d5dbe526769e245a4dcd905213b869894aa310b324dcc0fe8e8b6ec536f486f5f70444af51343d140e20b9e3ba38abba72f5f07481b120146cc9e0ab5b7a5f9d1ee9cfd0583819b206fb6cef68cdf0fb5691d5aa868e85810ae9fde6b7df245db383e3c3342be433051a87d951ac0b54ce88c656266ce24a1dceed9350247b1d4f28ed27cb4701a3771ef3"), []byte("odd")},
		{hexDecode("dd76b25b4ceffa6f22edcc82a82ab695eb92b46d136469d1a64d4f780036150b8ba38129bed6f1bc52af329f78dd5af46f475f854df8674c1da720ee1a915ae877246ae54e38bbbc4fcce05f2548184cd7c8e382643879a9411c63b96fd99826cec964533342668fa8b9bf3ea41e1e30e11832b005e294fd3e1c1d6e4fb9fd385131361e8db442c76d83dff8826330f226fc080fb7580ff8b49dc8afd1ad84b909eb2d8907979dba7f2907964c11ae808d5829a550b71d81fe145e77e4390038ef9e39c3c827247e329a90ea182f52742a39b087f7406f76bdd2d2baaf4e46eb013fa8cca4d8b0b3c7657a8bcdcd8d6ab5b7fa1ef5bd30b582545780a50d18c977014eb7fa8399ca08a407ea04aa18c3f90fdfb076f344302faffde3a82178c06c5c6450a3bf23c17282ad374903c334b17203bf00f680d5cebdbf280f856398e0d49a652be9ac6ff07300f1e24ed025432b6e4de0c2"), []byte("odd")},
		{hexDecode("dffd38f69b37bc3cbde413a43dbbad77d5785fa46d07553a721e99bf25a1721027dadd1911edc77263f791132510da085f7e15547a201fb69d0c1339cc0cedbd49d73d17"), []byte("odd")},
		{hexDecode("f9ab21e38bdfb19fc2e3131c68c066d9a60a9247571112f4d4c3e85e7a1bdc2a0fbe78c594af45a4d2f106eaf25e39de53cd2ea5ea6e201b5feb773bf72f74e8542d791b218700a5dc08b1d0954ccf243f0ad77e781c5ccd875c755022df58a1ef0682ad0deb088fb989c5f404decb6eb4fbe9e6559eb729441f54e012ed18fc81494b002a7470c378ae794efc46b323bcb8cb21eb9ef0433f1751431640704bda2bddf033b22a4281430f7bb81d9ab1f7a441b466a94a0539707395ba3dc19b56195deddd2e884a9587814aa494f8667c1c332d05374c9615d2b90171e444cdc5"), []byte("odd")},
		{hexDecode("fa554c4b9227db30b76c4eb6f276f12d13b4340de8621924e81eba74e21030592f4106cfc4898a9912f0b14eddce86cdc8c99e7477090994f586ec6f137a33b8acb593821a84e6a3b3c3ac2326f491f7fff00afd01d8dc5f297fac2a5040a5dccd3c9d14b2eb2d3ef7624759cf5c0e070e879a12050b59b410680ac3ab905091fac0dfaa37707219e6d155f4d24d8b9784f24db743f026f4e6f2f62d1f499dee0671b5a2b2c3ea9c91b6822c46a3b9de98558c21f9468b3d68ab11387771b38396a9e0f31adc6007dfb1d1cf47df87705702a093de14f1fdf88f854d2a2cd4de092077638438cb4dc3b4ff554d5d7a474fb1af8cccbb37bcfe42ff4378a353f6601676b9d18c83c8da1849ec4ba31ff2c85c9733988bf2b7fdb10147d7a4797951f2bacffa4c960221ac00bdfe7cc980c72614a6e0ae1c9c735c482e3a1555af424151bc465bf1bb5be30dff311533872ae4b4f3d4319224b210cf68e3d53830a3326a3a95f6f3cd539537528f03"), []byte("odd")},
		{hexDecode("fcae5cbfe5c5bb25bfdc5b6ff2acf09dcd6fa7f8eba466ca4f5e7c72ce4006b16087036c1b96f1b29768fddb7ebd27976cd29f5d6272348a384e36c878e706eff8bc1b8793ac3f02cd4ff6135a5dc4fcda1169dd3e6b412c44ced1898aeac15016eb81bb69c774ddbb91807dd1780180454c945d09514c25bba1aac19e4e2a24cc7fb74c5e45"), []byte("odd")},
	}

	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	for _, test := range rt {
		val, err := trie.Get(test.key)
		if err != nil {
			t.Errorf("Fail to get key %x: %s", test.key, err.Error())
		} else if !bytes.Equal(val, test.value) {
			t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
		}
	}
}

func TestGetPartialKey(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("pen")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("penguin")
	key3 := []byte{0xf2}
	value3 := []byte("feather")
	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	key5 := []byte{}
	value5 := []byte("floof")
	key6 := []byte{0x01, 0x35, 0x07}
	value6 := []byte("odd")

	pk0 := []byte{0x1, 0x3, 0x5}
	pk1 := []byte{0x3, 0x5}
	pk2 := []byte{0x9}
	pk3 := []byte{0x2}
	pk4 := []byte{0x0d, 0x03}

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	var val []byte
	leaf, err := trie.getLeaf(key2)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key2)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(leaf.value, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	} else if !bytes.Equal(leaf.key, pk2) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk2, leaf.key)
	}

	err = trie.Put(key6, value6)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	leaf, err = trie.getLeaf(key1)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key1)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(leaf.value, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	} else if !bytes.Equal(leaf.key, pk0) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk0, leaf.key)
	}

	leaf, err = trie.getLeaf(key2)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key2)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(leaf.value, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	} else if !bytes.Equal(leaf.key, nil) {
		t.Errorf("Fail to get correct partial key nil: got %x", leaf.key)
	}

	leaf, err = trie.getLeaf(key6)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key6)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key6, err.Error())
	} else if !bytes.Equal(leaf.value, value6) {
		t.Errorf("Fail to get key %x with value %x: got %x", key6, value6, val)
	} else if !bytes.Equal(leaf.key, nil) {
		t.Errorf("Fail to get correct partial key nil: got %x", leaf.key)
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key5, err.Error())
	} else if !bytes.Equal(val, value5) {
		t.Errorf("Fail to get key %x with value %x: got %x", key5, value5, val)
	}

	leaf, err = trie.getLeaf(key1)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key1)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(leaf.value, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	} else if !bytes.Equal(leaf.key, pk1) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk1, leaf.key)
	}

	leaf, err = trie.getLeaf(key2)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key2)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(leaf.value, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	} else if !bytes.Equal(leaf.key, nil) {
		t.Errorf("Fail to get correct partial key nil: got %x", leaf.key)
	}

	leaf, err = trie.getLeaf(key3)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key3)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(leaf.value, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	} else if !bytes.Equal(leaf.key, pk3) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk3, leaf.key)
	}

	leaf, err = trie.getLeaf(key4)
	if leaf == nil {
		t.Errorf("Fail to get key %x: nil leaf", key4)
	} else if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(leaf.value, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	} else if !bytes.Equal(leaf.key, pk4) {
		t.Errorf("Fail to get correct partial key %x: got %x", pk4, leaf.key)
	}
}

func buildSmallTrie() *Trie {
	trie := newEmpty()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("pen")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("penguin")
	key3 := []byte{0xf2}
	value3 := []byte("feather")
	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	key5 := []byte{}
	value5 := []byte("floof")
	key6 := []byte{0x01, 0x35, 0x07}
	value6 := []byte("odd")

	err := trie.Put(key1, value1)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key5, value5)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	err = trie.Put(key6, value6)
	if err != nil {
		log.Fatalf("Fail to put with key %x and value %x: %s", key6, value6, err.Error())
	}

	return trie
}
func TestDeleteSmall(t *testing.T) {
	trie := buildSmallTrie()

	key1 := []byte{0x01, 0x35}
	value1 := []byte("pen")
	key2 := []byte{0x01, 0x35, 0x79}
	value2 := []byte("penguin")
	key3 := []byte{0xf2}
	value3 := []byte("feather")
	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	key5 := []byte{}
	value5 := []byte("floof")
	key6 := []byte{0x01, 0x35, 0x07}
	value6 := []byte("odd")

	// key5 = nil
	err := trie.Delete(key5)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key5, err.Error())
	}

	val, err := trie.Get(key5)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key5, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key5, value5, val)
	}

	trie = buildSmallTrie()

	// key4 = 09d3
	err = trie.Delete(key4)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key4, err.Error())
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key4, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key4, value4, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	trie = buildSmallTrie()

	// key3 = f2
	err = trie.Delete(key3)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key3, err.Error())
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key3, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key3, value3, val)
	}

	trie = buildSmallTrie()

	// key5 = nil
	err = trie.Delete(key5)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key5, err.Error())
	}

	err = trie.Delete(key3)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key3, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key5, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key5, value5, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	trie = buildSmallTrie()

	// key2 = 013579
	err = trie.Delete(key2)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key2, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key2, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	trie = buildSmallTrie()

	// key2 = 0135
	err = trie.Delete(key1)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key1, err.Error())
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key1, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	trie = buildSmallTrie()

	// key6 = 0135
	err = trie.Delete(key6)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key6, err.Error())
	}

	val, err = trie.Get(key6)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key6, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key6, value6, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(value2, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(value1, val) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}
}

func TestCombineBranch(t *testing.T) {
	trie := buildSmallTrie()

	// key1 := []byte{0x01, 0x35}
	// value1 := []byte("pen")
	// key2 := []byte{0x01, 0x35, 0x79}
	// value2 := []byte("penguin")
	// key3 := []byte{0xf2}
	// value3 := []byte("feather")
	key4 := []byte{0x09, 0xd3}
	value4 := []byte("noot")
	// key5 := []byte{}
	// value5 := []byte("floof")
	key6 := []byte{0x01, 0x35, 0x46}
	value6 := []byte("raccoon")
	key7 := []byte{0x01, 0x35, 0x46, 0x77}
	value7 := []byte("rat")


	err := trie.Put(key6, value6)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key6, value6, err.Error())
	}

	err = trie.Put(key7, value7)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key7, value7, err.Error())
	}

	err = trie.Delete(key4)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key4, err.Error())
		t.Errorf("Fail to delete key %x: %s", key4, err.Error())
	}

	val, err := trie.Get(key4)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key4, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key4, value4, val)
	}

	}

func TestDeleteOddKeyLengths(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x43, 0xc1}
	value1 := []byte("noot")
	key2 := []byte{0x49, 0x29}
	value2 := []byte("nootagain")
	key3 := []byte{0x43, 0x0c}
	value3 := []byte("odd")
	key4 := []byte{0x4f, 0x4d}
	value4 := []byte("stuff")
	key5 := []byte{0xf4, 0xbc}
	value5 := []byte("spaghetti")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	val, err := trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	val, err = trie.Get(key5)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key5, err.Error())
	} else if !bytes.Equal(val, value5) {
		t.Errorf("Fail to get key %x with value %x: got %x", key5, value5, val)
	}
	//
	//err = trie.Delete(key1)
	//if err != nil {
	//	t.Errorf("Fail to delete key %x: %s", key1, err.Error())
	//}
	//
	//val, err = trie.Get(key1)
	//if err != nil {
	//	t.Errorf("Error when attempting to get deleted key %x: %s", key1, err.Error())
	//} else if val != nil {
	//	t.Errorf("Fail to delete key %x with value %x: got %x", key1, value1, val)
	//}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}
}

// To be used once trie.Delete is implemented
func TestDelete(t *testing.T) {
	trie := newEmpty()

	rt := generateRandTest(1000)
	for _, test := range rt {
		err := trie.Put(test.key, test.value)
		if err != nil {
			t.Errorf("Fail to put with key %x and value %x: %s", test.key, test.value, err.Error())
		}
	}

	for _, test := range rt {
		r := rand.Int() % 2
		switch r {
		case 0:
			//t.Logf("DEL %x", test.key)
			err := trie.Delete(test.key)
			if err != nil {
				t.Errorf("Fail to delete key %x: %s", test.key, err.Error())
				// for _, othertest := range rt {
				// 	if othertest.key[0] == test.key[0] {
				// 		t.Logf("%x", othertest.key)
				// 	}
				// }
			}

			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Error when attempting to get deleted key %x: %s", test.key, err.Error())
			} else if val != nil {
				t.Errorf("Fail to delete key %x with value %x: got %x", test.key, test.value, val)
			}
		case 1:
			val, err := trie.Get(test.key)
			if err != nil {
				t.Errorf("Error when attempting to get key %x: %s", test.key, err.Error())
			} else if !bytes.Equal(test.value, val) {
				t.Errorf("Fail to get key %x with value %x: got %x", test.key, test.value, val)
				// for _, othertest := range rt {
				// 	if othertest.key[0] == test.key[0] {
				// 		t.Logf("%x", othertest.key)
				// 	}
				// }
			}
		}
	}
}
//
//func TestDeleteFromBranch(t *testing.T) {
//	trie := newEmpty()
//
//	key1 := []byte{0x07, 0x7a}
//	value1 := []byte("noot")
//	key2 := []byte{0x07, 0x9c}
//	value2 := []byte("nootagain")
//	key3 := []byte{0x51, 0xb5}
//	value3 := []byte("odd")
//	key4 := []byte{0x51, 0xef}
//	value4 := []byte("stuff")
//
//	err := trie.Put(key1, value1)
//	if err != nil {
//		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
//	}
//
//	err = trie.Put(key2, value2)
//	if err != nil {
//		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
//	}
//
//	err = trie.Put(key3, value3)
//	if err != nil {
//		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
//	}
//
//	err = trie.Put(key4, value4)
//	if err != nil {
//		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
//	}
//
//	err = trie.Delete(key1)
//	if err != nil {
//		t.Errorf("Fail to delete key %x: %s", key1, err.Error())
//	}
//
//	val, err := trie.Get(key1)
//	if err != nil {
//		t.Errorf("Error when attempting to get deleted key %x: %s", key1, err.Error())
//	} else if val != nil {
//		t.Errorf("Fail to delete key %x with value %x: got %x", key1, value1, val)
//	}
//
//	val, err = trie.Get(key2)
//	if err != nil {
//		t.Errorf("Fail to get key %x: %s", key2, err.Error())
//	} else if !bytes.Equal(val, value2) {
//		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
//	}
//
//	err = trie.Delete(key3)
//	if err != nil {
//		t.Errorf("Fail to delete key %x: %s", key3, err.Error())
//	}
//
//	val, err = trie.Get(key3)
//	if err != nil {
//		t.Errorf("Error when attempting to get deleted key %x: %s", key3, err.Error())
//	} else if val != nil {
//		t.Errorf("Fail to delete key %x with value %x: got %x", key3, value3, val)
//	}
//
//	val, err = trie.Get(key4)
//	if err != nil {
//		t.Errorf("Fail to get key %x: %s", key4, err.Error())
//	} else if !bytes.Equal(val, value4) {
//		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
//	}
//}

func TestDeleteFromBranch(t *testing.T) {
	trie := newEmpty()

	key1 := []byte{0x06, 0x15, 0xfc}
	value1 := []byte("noot")
	key2 := []byte{0x06, 0x2b, 0xa9}
	value2 := []byte("nootagain")
	key3 := []byte{0x06, 0xaf, 0xb1}
	value3 := []byte("odd")
	key4 := []byte{0x06, 0xa3, 0xff}
	value4 := []byte("stuff")
	key5 := []byte{0x43, 0x21}
	value5 := []byte("stuffagain")

	err := trie.Put(key1, value1)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key1, value1, err.Error())
	}

	err = trie.Put(key2, value2)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key2, value2, err.Error())
	}

	err = trie.Put(key3, value3)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key3, value3, err.Error())
	}

	err = trie.Put(key4, value4)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key4, value4, err.Error())
	}

	err = trie.Put(key5, value5)
	if err != nil {
		t.Errorf("Fail to put with key %x and value %x: %s", key5, value5, err.Error())
	}

	val, err := trie.Get(key1)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key1, err.Error())
	} else if !bytes.Equal(val, value1) {
		t.Errorf("Fail to get key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Delete(key1)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key1, err.Error())
	}

	val, err = trie.Get(key1)
	if err != nil {
		t.Errorf("Error when attempting to get deleted key %x: %s", key1, err.Error())
	} else if val != nil {
		t.Errorf("Fail to delete key %x with value %x: got %x", key1, value1, val)
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key3)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key3, err.Error())
	} else if !bytes.Equal(val, value3) {
		t.Errorf("Fail to get key %x with value %x: got %x", key3, value3, val)
	}

	err = trie.Delete(key3)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key3, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}

	val, err = trie.Get(key4)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key4, err.Error())
	} else if !bytes.Equal(val, value4) {
		t.Errorf("Fail to get key %x with value %x: got %x", key4, value4, val)
	}

	err = trie.Delete(key4)
	if err != nil {
		t.Errorf("Fail to delete key %x: %s", key4, err.Error())
	}

	val, err = trie.Get(key2)
	if err != nil {
		t.Errorf("Fail to get key %x: %s", key2, err.Error())
	} else if !bytes.Equal(val, value2) {
		t.Errorf("Fail to get key %x with value %x: got %x", key2, value2, val)
	}
}