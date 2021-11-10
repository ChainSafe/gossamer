package pprof

// Settings are the settings for the Pprof service.
type Settings struct {
	// ListeningAddress is the HTTP pprof server
	// listening address.
	ListeningAddress string
	// See runtime.SetBlockProfileRate
	// Set to 0 to disable profiling.
	BlockProfileRate int
	// See runtime.SetMutexProfileFraction
	// Set to 0 to disable profiling.
	MutexProfileRate int
}

func (s *Settings) setDefaults() {
	if s.ListeningAddress == "" {
		s.ListeningAddress = "localhost:6060"
	}
}
