// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package runtime

import (
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/ChainSafe/log15"
)

// CustomFileHandler returns a Handler that adds the name of the calling function to the context with key "func"
//  and the line number and file of the calling function to the context with key "caller".
func CustomFileHandler(h log.Handler) log.Handler {
	return log.FuncHandler(func(r *log.Record) error {
		r.Ctx = append(r.Ctx, "func", strings.TrimLeft(filepath.Ext(r.Call.Frame().Function), "."), "caller", fmt.Sprint(r.Call))
		return h.Log(r)
	})
}
