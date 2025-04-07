package chainspec

import (
	"errors"

	"github.com/ChainSafe/gossamer/internal/client/executor"
	primitives_storage "github.com/ChainSafe/gossamer/internal/primitives/storage"
	"github.com/ChainSafe/gossamer/internal/primitives/storage/keys"
)

func ResolveStateVersionFromWasm[E executor.RuntimeVersionOf](
	storage primitives_storage.Storage,
	executor E,
) (primitives_storage.StateVersion, error) {
	_, has := storage.Top.Get(string(keys.Code))
	if !has {
		return primitives_storage.NoStateVersion,
			errors.New("runtime missing from initial storage, could not read state version")
	}

	panic("not implemented")
}
