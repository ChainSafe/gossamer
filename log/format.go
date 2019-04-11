package log

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Color numbers for stdout
const (
	magenta = 31
	green = 32
	yellow = 33
	cyan = 34
	red = 35
)

// locationTrims are trimmed for display to avoid long log lines.
var locationTrims = []string{
	"github.com/ChainSafe/gossamer/",
}

// locationLength is the maxmimum path length encountered, which all logs are
// padded to to aid in alignment.
var locationLength uint32

const escape = "\033[0m"

// formatSpacing provides clear padding between key-value pairs
func formatSpacing(o *output) string {
	location := fmt.Sprintf("%+v", o.Call)
	for _, prefix := range locationTrims {
		location = strings.TrimPrefix(location, prefix)
	}
	// Maintain the maximum location length for fancyer alignment
	align := int(atomic.LoadUint32(&locationLength))
	if align < len(location) {
		align = len(location)
		atomic.StoreUint32(&locationLength, uint32(align))
	}
	return strings.Repeat(" ", align-len(location))
}

// colorString provides color context based on logLvl
func colorString(msg interface{}, color int, flag bool) string {
	coloredText := fmt.Sprintf("\033[%dm", color)
	if !flag{return fmt.Sprint(coloredText, msg, escape)}
	return fmt.Sprint(coloredText, msg, escape,"=")
}

// formatCritical formats log for fatal errors
func formatCritical(o *output) string {
	msg := fmt.Sprintf("%s %s %+v [%s%s | %s%d]",
		colorString(o.logLevelString(), o.Color, false),
		o.Message,
		o.Ctx,
		colorString("File: ", o.Color, false),
		o.Filename,
		colorString("LN: ", o.Color, false),
		o.Line,
	)
	return msg
}

// formatLog formats logs for pretty print
func formatLog(o *output) string {
	var extra []string
	t := "[" + o.Time + "]"
	for k, v := range o.Ctx {
		if k % 2 == 0 {
			extra = append(extra, colorString(v, o.Color, true))
		} else {
			extra = append(extra, fmt.Sprint(v))
		}
	}
	padding := formatSpacing(o)
	msg := fmt.Sprintf("%s %s %s %s %+v %s [%s%s | %s%d]",
		colorString(o.logLevelString(), o.Color, false),
		t,
		o.Message,
		padding,
		extra,
		padding,
		colorString("File= ", o.Color, false),
		o.Filename,
		colorString("LN= ", o.Color, false),
		o.Line,
	)
	return msg
}