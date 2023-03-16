package newWasmer

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// version calls runtime function Core_Version and returns the
// decoded version structure.
func (in *Instance) version() (version runtime.Version, err error) {
	res, err := in.Exec(runtime.CoreVersion, []byte{})
	if err != nil {
		return version, err
	}

	version, err = runtime.DecodeVersion(res)
	if err != nil {
		return version, fmt.Errorf("decoding version: %w", err)
	}

	return version, nil
}
