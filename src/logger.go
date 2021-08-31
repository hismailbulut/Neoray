package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sqweek/dialog"
)

const ENABLE_COLORED_OUTPUT = true

const (
	AnsiBlack   = "\u001b[30m"
	AnsiRed     = "\u001b[31m"
	AnsiGreen   = "\u001b[32m"
	AnsiYellow  = "\u001b[33m"
	AnsiBlue    = "\u001b[34m"
	AnsiMagenta = "\u001b[35m"
	AnsiCyan    = "\u001b[36m"
	AnsiWhite   = "\u001b[37m"
	AnsiReset   = "\u001b[0m"
)

type LogLevel uint32
type LogType uint32

const (
	// log levels
	LOG_LEVEL_DEBUG LogLevel = iota
	LOG_LEVEL_TRACE
	LOG_LEVEL_WARN
	LOG_LEVEL_ERROR
	// The fatal will be printed to logfile and a
	// fatal popup will be shown. The program exits immediately.
	LOG_LEVEL_FATAL
)

const (
	// Log types, makes easy to understand where the message coming from
	LOG_TYPE_NVIM LogType = iota
	LOG_TYPE_NEORAY
	LOG_TYPE_RENDERER
	LOG_TYPE_PERFORMANCE
)

var (
	verboseFile *os.File = nil
)

func initVerboseFile(filename string) {
	if verboseFile != nil {
		return
	}
	path, err := filepath.Abs(filename)
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to get absolute path:", err)
		return
	}
	verboseFile, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0666)
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create log file:", err)
		return
	}
	// Print informations to log file.
	verboseFile.WriteString(fmt.Sprintln("\nNEORAY", versionString(), buildTypeString(), "LOG", time.Now().UTC()))
}

func shutdownLogger() {
	// If we are panicking print it to logfile.
	// Also the stack trace will be printed after
	// fatal error.
	if pmsg := recover(); pmsg != nil {
		// Create crash report.
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "[PANIC!]", pmsg)
	}
	cleanup()
}

// This function will always be called.
func cleanup() {
	// If logfile is initialized then close it.
	if verboseFile != nil {
		verboseFile.Close()
	}
	// Reset terminal color
	if ENABLE_COLORED_OUTPUT {
		fmt.Print(AnsiReset)
	}
}

func createCrashReport(msg string) {
	crash_file, err := os.Create("neoray_crash.log")
	if err == nil {
		defer crash_file.Close()
		crash_file.WriteString(fmt.Sprintln("NEORAY", versionString(), buildTypeString(), "Crash Report", time.Now().UTC()))
		crash_file.WriteString("Please open an issue in github with this file.\n")
		crash_file.WriteString("The program is crashed because of the following reasons:\n")
		crash_file.WriteString(msg)
		crash_file.WriteString("\ngoroutine dump:\n")
		stackTrace := make([]byte, 1<<15)
		stackLen := runtime.Stack(stackTrace, true)
		crash_file.WriteString(string(stackTrace[:stackLen]))
	}
}

func logMessage(log_level LogLevel, log_type LogType, message ...interface{}) {
	if log_level < MINIMUM_LOG_LEVEL && verboseFile == nil {
		return
	}

	fatal := false
	colorCode := ""
	logLevelString := ""
	switch log_level {
	case LOG_LEVEL_DEBUG:
		logLevelString = "[DEBUG]"
		colorCode = AnsiWhite
	case LOG_LEVEL_TRACE:
		logLevelString = "[TRACE]"
		colorCode = AnsiGreen
	case LOG_LEVEL_WARN:
		logLevelString = "[WARNING]"
		colorCode = AnsiYellow
	case LOG_LEVEL_ERROR:
		logLevelString = "[ERROR]"
		colorCode = AnsiRed
	case LOG_LEVEL_FATAL:
		logLevelString = "[FATAL]"
		colorCode = AnsiRed
		fatal = true
	default:
		panic("invalid log level")
	}

	logTypeString := ""
	switch log_type {
	case LOG_TYPE_NVIM:
		logTypeString = "[NVIM]"
	case LOG_TYPE_NEORAY:
		logTypeString = "[NEORAY]"
	case LOG_TYPE_RENDERER:
		logTypeString = "[RENDERER]"
	case LOG_TYPE_PERFORMANCE:
		logTypeString = "[PERFORMANCE]"
	default:
		panic("invalid log type")
	}

	logString := logLevelString + " " + logTypeString + " " + fmt.Sprintln(message...)

	// Print to stdout
	if ENABLE_COLORED_OUTPUT {
		fmt.Print(colorCode + logString)
	} else {
		fmt.Print(logString)
	}

	// Print to verbose file if opened
	if verboseFile != nil {
		verboseFile.WriteString(logString)
	}

	if fatal {
		// Create crash report file
		createCrashReport(logString)
		// Open message box with error
		dialog.Message(logString).Error()
		// Cleanup and shutdown.
		cleanup()
		os.Exit(1)
	}
}

// Fast debug message
func logDebug(message ...interface{}) {
	logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, message...)
}

// Fast debug message using format
func logfDebug(format string, message ...interface{}) {
	logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, fmt.Sprintf(format, message...))
}

// Debug message with type
func logDebugMsg(log_type LogType, message ...interface{}) {
	logMessage(LOG_LEVEL_DEBUG, log_type, message...)
}

// This assert logs fatal when cond is false.
func assert(cond bool, message ...interface{}) {
	if cond == false {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Assertion Failed:", fmt.Sprint(message...))
	}
}

// This assert logs error when cond is false.
func assert_error(cond bool, message ...interface{}) {
	if cond == false {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Assertion Failed:", fmt.Sprint(message...))
	}
}
