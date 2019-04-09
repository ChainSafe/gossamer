package log

// Import packages
import (
	"fmt"
	"path"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	// Map for the various codes of colors
	colors map[LogLevel]string

	// Contains color strings for stdout
	logNo uint64
)

// LogLevel type
type LogLevel int

// Color numbers for stdout
const (
	red = iota + 30
	green
	yellow
	magenta
	cyan
)

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
}

type Logger interface {
	Critical(msg string, extra ...interface{})
	Info(msg string, extra ...interface{})
	Warn(msg string, extra ...interface{})
	Err(msg string, extra ...interface{})
	Debug(msg string, extra ...interface{})
}

type logger struct {
	write  []interface{}
}

// init pkg
func init() {
	initColors()
}

func (l *logger) Critical(msg string, a ...interface{}) {
	l.Output(criticalLvl, msg, 2)
}

// Info logs a message at Info level
func (l *logger) Info(msg string, a ...interface{}) {
	l.Output(infoLvl, msg, 2)
}

// Returns a proper string to output for colored logging
func colorString(color int) string {
	return fmt.Sprintf("\033[%dm", int(color))
}

// Initializes the map of colors
func initColors() {
	colors = map[LogLevel]string{
		criticalLvl: colorString(magenta),
		errLvl:    colorString(red),
		warnLvl:  colorString(yellow),
		debugLvl:    colorString(cyan),
		infoLvl:     colorString(green),
	}
}

func (l *logger) Output(lvl LogLevel, message string, pos int) string {
	_, filename, line, _ := runtime.Caller(pos)
	filename = path.Base(filename)
	o:= &output{
		Id:       atomic.AddUint64(&logNo, 1),
		Time:     time.Now().Format("yyyy-mm-dd hh:mm:ss"),
		Level:    lvl,
		Message:  message,
		Filename: filename,
		Line:     line,
	}
	msg := fmt.Sprintf(o.Message,
		o.Id,
		o.Time,
		o.Filename,
		o.Line,
		o.logLevelString(),
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
		"WARNING",
		"NOTICE",
		"DEBUG",
		"INFO",
	}
	return logLevels[o.Level-1]
}




//// InfoF logs a message at Info level using the same syntax and options as fmt.Printf
//func (l *Logger) InfoF(format string, a ...interface{}) {
//	l.log_internal(Info, fmt.Sprintf(format, a...), 2)
//}
//
//// Debug logs a message at Debug level
//func (l *Logger) Debug(message string) {
//	l.log_internal(Debug, message, 2)
//}
//
//// DebugF logs a message at Debug level using the same syntax and options as fmt.Printf
//func (l *Logger) DebugF(format string, a ...interface{}) {
//	l.log_internal(Debug, fmt.Sprintf(format, a...), 2)
//}





//// CriticalF logs a message at Critical level using the same syntax and options as fmt.Printf
//func (l *Logger) Criticalf(format string, a ...interface{}) {
//	l.log_internal(Critical, fmt.Sprintf(format, a...), 2)
//}
//
//// Error logs a message at Error level
//func (l *Logger) Error(message string) {
//	l.log_internal(Err, message, 2)
//}
//
//// ErrorF logs a message at Error level using the same syntax and options as fmt.Printf
//func (l *Logger) ErrorF(format string, a ...interface{}) {
//	l.log_internal(Err, fmt.Sprintf(format, a...), 2)
//}
//
//// Warning logs a message at Warning level
//func (l *Logger) Warning(message string) {
//	l.log_internal(Warn, message, 2)
//}
//
//// WarningF logs a message at Warning level using the same syntax and options as fmt.Printf
//func (l *Logger) WarningF(format string, a ...interface{}) {
//	l.log_internal(Warn, fmt.Sprintf(format, a...), 2)
//}

