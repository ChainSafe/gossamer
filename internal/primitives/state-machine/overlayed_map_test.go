package statemachine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCloneOverlayedMap(t *testing.T) {
	overlayed := NewOverlayedMap[string, string]()
	key := "key1"
	overlayed.SetOffchain(key, "value1", nil)

	cloned := overlayed.Clone()

	require.Equal(t, overlayed.Get(key), cloned.Get(key))

	cloned.SetOffchain(key, "value2", nil)
	require.NotEqual(t, overlayed.Get(key), cloned.Get(key))
}
