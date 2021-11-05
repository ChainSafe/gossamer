package httpserver

// Logger is the logger interface accepted by the
// HTTP server.
type Logger interface {
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
}
