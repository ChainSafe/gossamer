package log

// Import packages
import (
	"fmt"
	"log"
	"path"
	"runtime"
	"strings"
	"time"
)

// LogLevel type
type LogLevel int

// Color numbers for stdout
const (
	magenta = 31
	green = 32
	yellow = 33
	cyan = 34
	red = 35
)

const escape = "\033[0m"

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
}

type Logger interface {
	Critical(msg string, extra ...interface{})
	Info(msg string, extra ...interface{})
	Warn(msg string, extra ...interface{})
	Err(msg string, extra ...interface{})
	Debug(msg string, extra ...interface{})
}

// Info logs a message at Info level
func InfoV(msg string, extra ...interface{}) {
	fmt.Println(OutputV(infoLvl, msg, extra, 2, green))
}

func WarnV(msg string, extra ...interface{}) {
	fmt.Println(OutputV(warnLvl, msg, extra, 2, yellow))
}

func DebugV(msg string, extra ...interface{}) {
	fmt.Println(OutputV(debugLvl, msg, extra, 2, cyan))
}

func CriticalV(msg string, extra ...interface{}) {
	log.Fatal(OutputV(criticalLvl, msg, extra, 2, red))
}

func ErrV(msg string, extra ...interface{}) {
	fmt.Println(OutputV(errLvl, msg, extra, 2, magenta))
}

func colorString(lvl interface{}, color int) string {
	coloredText := fmt.Sprintf("\033[%dm", color)
	return fmt.Sprint(coloredText, lvl, escape)
}

func OutputV(lvl LogLevel, message string, ctx []interface{}, pos, color int) string {
	_, filename, line, _ := runtime.Caller(pos)
	filename = path.Base(filename)
	o:= &output{
		Time:     time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
		Level:    lvl,
		Message:  message,
		Ctx: 	  ctx,
		Filename: filename,
		Line:     line,
		Color:	  color,
	}
	return checkLogV(o)
}

func checkLogV(o *output) string {
	if o.Color == 35 {
		return formatCriticalV(o)
	}
	if len(o.Ctx) != 0 {
		return formatLogV(o)
	}
	t := "[" + o.Time + "]"
	padding := strings.Repeat(" ", 10)

	msg := fmt.Sprintf("%s %s %s %+v [%s%s | %s%d]",
		colorString(o.logLevelString(), o.Color),
		t,
		o.Message,
		padding,
		colorString("File: ", o.Color),
		o.Filename,
		colorString("LN: ", o.Color),
		o.Line,
	)
	return msg
}

func formatLogV(o *output) string {
	t := "[" + o.Time + "]"
	padding := strings.Repeat(" ", 10)

	msg := fmt.Sprintf("%s %s %s %s %+v [%s%s | %s%d]",
		colorString(o.logLevelString(), o.Color),
		t,
		o.Message,
		o.Ctx,
		padding,
		colorString("File: ", o.Color),
		o.Filename,
		colorString("LN: ", o.Color),
		o.Line,
	)
	return msg
}

func formatCriticalV(o *output) string {
	padding := strings.Repeat(" ", 10)

	msg := fmt.Sprintf("%s %s %+v %s [%s%s | %s%d]",
		colorString(o.logLevelString(), o.Color),
		o.Message,
		o.Ctx,
		padding,
		colorString("File: ", o.Color),
		o.Filename,
		colorString("LN: ", o.Color),
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