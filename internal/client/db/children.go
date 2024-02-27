package db

import (
	"fmt"

	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/database"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

func readChildren[K, V comparable](
	db database.Database[hash.H256], column uint32, prefix []byte, parentHash K,
) ([]V, error) {
	buf := prefix
	encoded := scale.MustMarshal(parentHash)
	buf = append(buf, encoded...)

	rawValOpt := db.Get(database.ColumnID(column), buf)

	if rawValOpt == nil {
		return nil, nil
	}
	rawVal := *rawValOpt

	var children []V
	err := scale.Unmarshal(rawVal, &children)
	if err != nil {
		return nil, fmt.Errorf("Error decoding children: %w", err)
	}

	return children, nil
}
