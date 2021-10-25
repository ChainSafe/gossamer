package log

// Patch resets all the settings of the logger and use the options given to
// set them. This is thread safe. This does not affect child loggers.
func (l *Logger) Patch(options ...Option) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.settings = newSettings(options)
}
