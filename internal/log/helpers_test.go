package log

// RFC3339 format
const timePrefixRegex = `^2[0-9]{3}-[0-1][0-9]-[0-3][0-9]T[0-2][0-9]:[0-5][0-9]:[0-5][0-9]Z `

func levelPtr(l Level) *Level { return &l }

func formatPtr(f Format) *Format { return &f }

func newCallerSettings(file, line, funC bool) callerSettings {
	return callerSettings{
		file: &file,
		line: &line,
		funC: &funC,
	}
}
