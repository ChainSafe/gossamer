// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package log

// RFC3339 format
const timePrefixRegex = `([0-9]+)-` +
	`(0[1-9]|1[012])-` +
	`(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3])` +
	`:([0-5][0-9])` +
	`:([0-5][0-9]|60)(\.[0-9]+)?(([Zz])|([\+|\-]([01][0-9]|2[0-3])` +
	`:[0-5][0-9])) `

func levelPtr(l Level) *Level { return &l }

func formatPtr(f Format) *Format { return &f }

func newCallerSettings(file, line, funC bool) callerSettings {
	return callerSettings{
		file: &file,
		line: &line,
		funC: &funC,
	}
}
