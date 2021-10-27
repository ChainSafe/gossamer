package log

// Patch resets all the settings of the logger and use the options given to
// set them. This is thread safe. This does not affect child loggers.
func (l *Logger) Patch(options ...Option) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.settings = newSettings(options)
}

// PatchLevel changes the logger level. This is thread safe.
// This does not affect child loggers.
func (l *Logger) PatchLevel(level Level) {
	if level == LevelDoNotChange {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.settings.level = &level
}

// PatchCallerFunc patches the logger caller function logging.
// This is thread safe and does not affect child loggers.
func (l *Logger) PatchCallerFunc(enabled bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.settings.caller.funC = &enabled
}
