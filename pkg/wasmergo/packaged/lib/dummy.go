// See https://github.com/golang/go/issues/26366.
package lib

import (
	_ "github.com/ChainSafe/gossamer/pkg/wasmergo/packaged/lib/darwin-aarch64"
	_ "github.com/ChainSafe/gossamer/pkg/wasmergo/packaged/lib/darwin-amd64"
	_ "github.com/ChainSafe/gossamer/pkg/wasmergo/packaged/lib/linux-aarch64"
	_ "github.com/ChainSafe/gossamer/pkg/wasmergo/packaged/lib/linux-amd64"
)
