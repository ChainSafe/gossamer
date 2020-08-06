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
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/ChainSafe/log15"
)

const (
	termTimeFormat = "01-02|15:04:05"
	termMsgJust    = 40
	errorKey       = "LOG15_ERROR"
	timeFormat     = "2006-01-02T15:04:05-0700"
	floatFormat    = 'f'
)

// TerminalFormatWLine formats log records optimized for human readability on
// a terminal with color-coded level output, terser human friendly timestamp and
// final file name element and line number for level WARN, ERROR or CRIT.
// This format should only be used for interactive programs or while developing.
//
//     [TIME] [LEVEL] (file/line) MESSAGE key=value key=value ...
//
// Example:
//
//     [May 16 20:58:45] [DBUG] remove route ns=haproxy addr=127.0.0.1:50002
//
func TerminalFormatWLine() log.Format {
	return log.FormatFunc(func(r *log.Record) []byte {
		var color = 0
		var includeLine = false
		switch r.Lvl {
		case log.LvlCrit:
			color = 35
			includeLine = true
		case log.LvlError:
			color = 31
			includeLine = true
		case log.LvlWarn:
			color = 33
			includeLine = true
		case log.LvlInfo:
			color = 32
		case log.LvlDebug:
			color = 36
		case log.LvlTrace:
			color = 34
		}
		b := &bytes.Buffer{}
		lvl := strings.ToUpper(r.Lvl.String())
		if color > 0 {
			if includeLine {
				fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] (%v) %s ", color, lvl, r.Time.Format(termTimeFormat), r.Call, r.Msg)
			} else {
				fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s] %s ", color, lvl, r.Time.Format(termTimeFormat), r.Msg)
			}
		} else {
			if includeLine {
				fmt.Fprintf(b, "[%s] [%s] [%v] %s ", lvl, r.Time.Format(termTimeFormat), r.Call, r.Msg)
			} else {
				fmt.Fprintf(b, "[%s] [%s] %s ", lvl, r.Time.Format(termTimeFormat), r.Msg)
			}

		}

		// try to justify the log output for short messages
		ml := len(r.Msg)
		if includeLine {
			ml = ml + len(r.Call.String())
		}
		if len(r.Ctx) > 0 && ml < termMsgJust {
			b.Write(bytes.Repeat([]byte{' '}, termMsgJust-ml))  //nolint
		}

		// print the keys logfmt style
		logfmt(b, r.Ctx, color)
		return b.Bytes()
	})
}

func logfmt(buf *bytes.Buffer, ctx []interface{}, color int) {
	for i := 0; i < len(ctx); i += 2 {
		if i != 0 {
			buf.WriteByte(' ')  //nolint
		}

		k, ok := ctx[i].(string)
		v := formatLogfmtValue(ctx[i+1])
		if !ok {
			k, v = errorKey, formatLogfmtValue(k)
		}

		// XXX: we should probably check that all of your key bytes aren't invalid
		if color > 0 {
			fmt.Fprintf(buf, "\x1b[%dm%s\x1b[0m=%s", color, k, v)
		} else {
			buf.WriteString(k)  //nolint
			buf.WriteByte('=')  //nolint
			buf.WriteString(v)  //nolint
		}
	}

	buf.WriteByte('\n')  //nolint
}

// formatValue formats a value for serialization
func formatLogfmtValue(value interface{}) string {
	if value == nil {
		return "nil"
	}

	if t, ok := value.(time.Time); ok {
		// Performance optimization: No need for escaping since the provided
		// timeFormat doesn't have any escape characters, and escaping is
		// expensive.
		return t.Format(timeFormat)
	}
	value = formatShared(value)
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), floatFormat, 3, 64)
	case float64:
		return strconv.FormatFloat(v, floatFormat, 3, 64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", value)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}

func formatShared(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		return v.Format(timeFormat)

	case error:
		return v.Error()

	case fmt.Stringer:
		return v.String()

	default:
		return v
	}
}

var stringBufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func escapeString(s string) string {
	needsQuotes := false
	needsEscape := false
	for _, r := range s {
		if r <= ' ' || r == '=' || r == '"' {
			needsQuotes = true
		}
		if r == '\\' || r == '"' || r == '\n' || r == '\r' || r == '\t' {
			needsEscape = true
		}
	}
	if !needsEscape && !needsQuotes {
		return s
	}
	e := stringBufPool.Get().(*bytes.Buffer)
	e.WriteByte('"')  //nolint
	for _, r := range s {
		switch r {
		case '\\', '"':
			e.WriteByte('\\')  //nolint
			e.WriteByte(byte(r))  //nolint
		case '\n':
			e.WriteString("\\n")  //nolint
		case '\r':
			e.WriteString("\\r")  //nolint
		case '\t':
			e.WriteString("\\t")  //nolint
		default:
			e.WriteRune(r)  //nolint
		}
	}
	e.WriteByte('"')  //nolint
	var ret string
	if needsQuotes {
		ret = e.String()
	} else {
		ret = string(e.Bytes()[1 : e.Len()-1])
	}
	e.Reset()
	stringBufPool.Put(e)
	return ret
}
