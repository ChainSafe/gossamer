// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

type callerSettings struct {
	file *bool
	line *bool
	funC *bool
}

func (c *callerSettings) mergeWith(other callerSettings) {
	if other.file != nil {
		value := *other.file
		c.file = &value
	}

	if other.line != nil {
		value := *other.line
		c.line = &value
	}

	if other.funC != nil {
		value := *other.funC
		c.funC = &value
	}

	// Keep depth to 1
}

func (c *callerSettings) setDefaults() {
	if c.file == nil {
		value := false
		c.file = &value
	}

	if c.line == nil {
		value := false
		c.line = &value
	}

	if c.funC == nil {
		value := false
		c.funC = &value
	}
}

func getCallerString(settings callerSettings) (s string) {
	if !*settings.file && !*settings.line && !*settings.funC {
		return ""
	}

	const depth = 3
	pc, file, line, ok := runtime.Caller(depth)
	if !ok {
		return "error"
	}

	var fields []string

	if *settings.file {
		fields = append(fields, filepath.Base(file))
	}

	if *settings.line {
		fields = append(fields, "L"+fmt.Sprint(line))
	}

	if *settings.funC {
		details := runtime.FuncForPC(pc)
		if details != nil {
			funcName := strings.TrimLeft(filepath.Ext(details.Name()), ".")
			fields = append(fields, funcName)
		}
	}

	return strings.Join(fields, ":")
}
