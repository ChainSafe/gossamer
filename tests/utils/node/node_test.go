package node

import (
	"context"
	"testing"
	"time"
)

func Test_Node_InitAndStartTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)

	n := New(t)

	n.InitAndStartTest(ctx, t, cancel)

	cancel()
}
