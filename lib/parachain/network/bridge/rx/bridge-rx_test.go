package rx

import (
	"fmt"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/parachain"
	"testing"
)

func TestSendOurViewUponConnection(t *testing.T) {
	view := parachain.View{
		Heads: []common.Hash{{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
			1, 1, 1}},
	}
	fmt.Printf("View %v\n", view)
}
