package newWasmer

import (
	"github.com/ChainSafe/gossamer/lib/runtime"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
	"github.com/wasmerio/wasmer-go/wasmer"
	"testing"
	"time"
)

var testChildKey = []byte("childKey")
var testKey = []byte("key")
var testValue = []byte("value")

func Test_ext_offchain_timestamp_version_1(t *testing.T) {
	inst := NewTestInstance(t, runtime.POLKADOT_RUNTIME_v0929)
	//runtimeFunc, ok := inst.vm.Exports["rtm_ext_offchain_timestamp_version_1"]
	runtimeFunc, err := inst.vm.Exports.GetFunction("rtm_ext_offchain_timestamp_version_1")
	require.NoError(t, err)

	res, err := runtimeFunc(0, 0)
	require.NoError(t, err)

	wasmRes := wasmer.NewI64(res)
	outputPtr, outputLength := splitPointerSize(wasmRes.I64())
	memory := inst.ctx.Memory.Data()
	data := memory[outputPtr : outputPtr+outputLength]
	var timestamp int64
	err = scale.Unmarshal(data, &timestamp)
	require.NoError(t, err)

	expected := time.Now().Unix()
	require.GreaterOrEqual(t, expected, timestamp)
}
