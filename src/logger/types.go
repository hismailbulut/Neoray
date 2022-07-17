package logger

import "fmt"

const UNKNOWN = "UNKNOWN"

type BuildType uint8

func (buildType BuildType) String() string {
	switch buildType {
	case DebugBuild:
		return "Debug"
	case ReleaseBuild:
		return "Release"
	}
	return UNKNOWN
}

const (
	DebugBuild BuildType = iota
	ReleaseBuild
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (version Version) String() string {
	return fmt.Sprintf("v%d.%d.%d", version.Major, version.Minor, version.Patch)
}

type AnsiTermColor string

const (
	AnsiBlack   AnsiTermColor = "\u001b[30m"
	AnsiRed     AnsiTermColor = "\u001b[31m"
	AnsiGreen   AnsiTermColor = "\u001b[32m"
	AnsiYellow  AnsiTermColor = "\u001b[33m"
	AnsiBlue    AnsiTermColor = "\u001b[34m"
	AnsiMagenta AnsiTermColor = "\u001b[35m"
	AnsiCyan    AnsiTermColor = "\u001b[36m"
	AnsiWhite   AnsiTermColor = "\u001b[37m"
	AnsiReset   AnsiTermColor = "\u001b[0m"
)

type LogLevel uint32

const (
	// log levels
	DEBUG LogLevel = iota
	TRACE
	WARN
	ERROR
	FATAL
)

func (logLevel LogLevel) String() string {
	switch logLevel {
	case DEBUG:
		return "[DEBUG]"
	case TRACE:
		return "[TRACE]"
	case WARN:
		return "[WARNING]"
	case ERROR:
		return "[ERROR]"
	case FATAL:
		return "[FATAL]"
	}
	return UNKNOWN
}

func (logLevel LogLevel) Color() AnsiTermColor {
	switch logLevel {
	case DEBUG:
		return AnsiWhite
	case TRACE:
		return AnsiGreen
	case WARN:
		return AnsiYellow
	case ERROR:
		return AnsiRed
	case FATAL:
		return AnsiRed
	}
	return UNKNOWN
}
