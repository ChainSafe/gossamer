// Copyright 2020 ChainSafe Systems (ON) Corp.
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

package common

import (
	"bytes"
	"testing"

	log "github.com/ChainSafe/log15"
	"github.com/stretchr/testify/require"
)

func testFormatter(f log.Format) (log.Logger, *bytes.Buffer) {
	l := log.New()
	var buf bytes.Buffer
	l.SetHandler(log.StreamHandler(&buf, f))
	return l, &buf
}

func TestTerminalFormatWLine(t *testing.T) {
	l, buf := testFormatter(TerminalFormatWLine())
	l.Error("some message", "x", 1, "y", 3.2, "equals", "=", "quote", "\"",
		"carriage_return", "bang"+string('\r')+"foo", "tab", "bar	baz", "newline", "foo\nbar")

	// skip timestamp in comparison
	got := buf.Bytes()[30:buf.Len()]
	expected := []byte(`(logging_test.go:36) some message           [31mx[0m=1 [31my[0m=3.200 [31mequals[0m="=" [31mquote[0m="\"" [31mcarriage_return[0m="bang\rfoo" [31mtab[0m="bar\tbaz" [31mnewline[0m="foo\nbar"` + "\n")
	require.Equal(t, expected, got)
}
