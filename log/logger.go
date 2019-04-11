package log

import (
	"fmt"
	"log"
	"path"
	"runtime"
	"time"
	"github.com/go-stack/stack"
)

const (
	timeFormat     = "2006-01-02T15:04:05-0700"
)

// LogLevel type
type LogLevel int

// Log Level
const (
	criticalLvl LogLevel = iota + 1
	errLvl
	warnLvl
	debugLvl
	infoLvl
)

type output struct {
	Id       uint64
	Time     string
	Level    LogLevel
	Line     int
	Filename string
	Message  string
	Color 	 int
	Ctx  []interface{}
	Call     stack.Call
}

// Info logs a message at Info level
func Info(msg string, extra ...interface{}) {
	fmt.Println(Output(infoLvl, msg, extra, 2, green))
}
// Warn logs a message at Info level
func Warn(msg string, extra ...interface{}) {
	fmt.Println(Output(warnLvl, msg, extra, 2, yellow))
}
// Debug logs a message at Info level
func Debug(msg string, extra ...interface{}) {
	fmt.Println(Output(debugLvl, msg, extra, 2, cyan))
}
// Critical logs a message at Info level
func Critical(msg string, extra ...interface{}) {
	log.Fatal(Output(criticalLvl, msg, extra, 2, red))
}
// Err logs a message at Info level
func Err(msg string, extra ...interface{}) {
	fmt.Println(Output(errLvl, msg, extra, 2, magenta))
}

// Output sets the output type with given fields and gets file and line where log is called from
func Output(lvl LogLevel, message string, ctx []interface{}, pos, color int) string {
	_, filename, line, _ := runtime.Caller(pos)
	filename = path.Base(filename)
	o:= &output{
		Time:     time.Now().Format(timeFormat),
		Level:    lvl,
		Message:  message,
		Ctx: 	  ctx,
		Filename: filename,
		Line:     line,
		Color:	  color,
	}
	return checkLog(o)
}

// checkLog determines checks the logLvl for CRITICAL and if additional key-value pairs were included
func checkLog(o *output) string {
	if o.Level == 1 {
		return formatCritical(o)
	}
	if len(o.Ctx) != 0 {
		return formatLog(o)
	}
	t := "[" + o.Time + "]"

	msg := fmt.Sprintf("%s %s %+v [%s%s | %s%d]",
		colorString(o.logLevelString(), o.Color, false),
		t,
		o.Message,
		colorString("File: ", o.Color, false),
		o.Filename,
		colorString("LN: ", o.Color, false),
		o.Line,
	)
	return msg
}

// Returns a string with the execution stack for this goroutine
func Stack() string {
	buf := make([]byte, 1000000)
	runtime.Stack(buf, false)
	return string(buf)
}

// Returns the loglevel as string
func (o *output) logLevelString() string {
	logLevels := [...]string{
		"CRITICAL",
		"ERROR",
		"WARN",
		"DEBUG",
		"INFO",
	}
	return logLevels[o.Level-1]
}