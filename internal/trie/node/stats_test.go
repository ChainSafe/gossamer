package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewStats(t *testing.T) {
	t.Parallel()

	const descendants uint32 = 10
	stats := NewStats(descendants)

	expected := Stats{
		Descendants: descendants,
	}
	assert.Equal(t, expected, stats)
}

func Test_Branch_GetDescendants(t *testing.T) {
	t.Parallel()

	const descendants uint32 = 10
	branch := &Branch{
		Stats: Stats{
			Descendants: descendants,
		},
	}
	result := branch.GetDescendants()

	assert.Equal(t, descendants, result)
}

func Test_Branch_AddDescendants(t *testing.T) {
	t.Parallel()

	const (
		initialDescendants uint32 = 10
		addDescendants     uint32 = 2
		finalDescendants   uint32 = 12
	)
	branch := &Branch{
		Stats: Stats{
			Descendants: initialDescendants,
		},
	}
	branch.AddDescendants(addDescendants)
	expected := &Branch{
		Stats: Stats{
			Descendants: finalDescendants,
		},
	}

	assert.Equal(t, expected, branch)
}

func Test_Branch_SubDescendants(t *testing.T) {
	t.Parallel()

	const (
		initialDescendants uint32 = 10
		subDescendants     uint32 = 2
		finalDescendants   uint32 = 8
	)
	branch := &Branch{
		Stats: Stats{
			Descendants: initialDescendants,
		},
	}
	branch.SubDescendants(subDescendants)
	expected := &Branch{
		Stats: Stats{
			Descendants: finalDescendants,
		},
	}

	assert.Equal(t, expected, branch)
}
