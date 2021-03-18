package wasmer

import (
	"io/ioutil"
	"testing"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func Test_TestWasm(t *testing.T) {
	t.Skip("remove this later")
	testWasm, err := ioutil.ReadFile("../../../test_wasm_with_memory.wasm")
	require.NoError(t, err)

	cfg := &Config{
		Imports: ImportsNodeRuntime,
	}
	cfg.LogLvl = log.LvlTrace

	in, err := NewInstance(testWasm, cfg)
	require.NoError(t, err)

	data := []byte("helloworld")

	ptr, err := in.malloc(uint32(len(data)))
	require.NoError(t, err)
	defer in.clear()

	// Store the data into memory
	in.store(data, int32(ptr))
	datalen := int32(len(data))

	runtimeFunc, err := in.vm.Exports.GetFunction("test_ext_blake2_256")
	require.NoError(t, err)

	//dataSpan := pointerAndSizeToInt64(int32(ptr), datalen)

	res, err := runtimeFunc(int32(ptr), datalen)
	require.NoError(t, err)

	outPtr := res.(int32)
	hash := in.load(outPtr, 32)
	t.Log(hash)

	// offset, length := int64ToPointerAndSize(res.(int64)) // TODO: are all returns int64?
	// ret := in.load(offset, length)
	// t.Log(ret)
}
