package log

// Patch patches the existing settings with any option given.
// This is thread safe and does not affect child loggers.
func (l *Logger) Patch(options ...Option) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	var updatedSettings settings
	updatedSettings.mergeWith(l.settings)
	updatedSettings.mergeWith(newSettings(options))
	l.settings = updatedSettings
}
