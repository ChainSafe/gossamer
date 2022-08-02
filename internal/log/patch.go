// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

// Patch patches the existing settings with any option given.
// This is thread safe and propagates to all child loggers.
// TODO-1946 remove patch progagation to child loggers.
func (l *Logger) Patch(options ...Option) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.patchWithoutLocking(options...)
	for _, child := range l.childs {
		child.patchWithoutLocking(options...)
	}
}

func (l *Logger) patchWithoutLocking(options ...Option) {
	var updatedSettings settings
	updatedSettings.mergeWith(l.settings)
	updatedSettings.mergeWith(newSettings(options))
	l.settings = updatedSettings
}
