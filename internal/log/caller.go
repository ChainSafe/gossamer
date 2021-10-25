package log

// Caller is the configuration to show the caller of the log.
type Caller uint8

const (
	// CallerHidden signals no caller should be logged.
	CallerHidden Caller = iota
	// CallerShort signals the short notation of the caller should be logged.
	CallerShort
)
