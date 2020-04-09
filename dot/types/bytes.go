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

package types

import (
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/scale"
)

// decodeOptionalBytes decodes SCALE encoded optional bytes into an *optional.Bytes
func decodeOptionalBytes(r io.Reader) (*optional.Bytes, error) {
	sd := scale.Decoder{Reader: r}

	exists, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	if exists == 1 {
		b, err := sd.Decode([]byte{})
		if err != nil {
			return nil, err
		}

		return optional.NewBytes(true, b.([]byte)), nil
	}

	return optional.NewBytes(false, nil), nil
}
