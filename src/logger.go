package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sqweek/dialog"
)

const enableAsciiColorOutput = true

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
	LEVEL_DEBUG LogLevel = iota
	LEVEL_TRACE
	LEVEL_WARN
	LEVEL_ERROR
	// The fatal will be printed to logfile and a
	// fatal popup will be shown. The program exits immediately.
	LEVEL_FATAL
)

const (
	// Log types, makes easy to understand where the message coming from
	TYPE_NVIM LogType = iota
	TYPE_NEORAY
	TYPE_RENDERER
	TYPE_PERFORMANCE
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
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to get absolute path:", err)
		return
	}
	verboseFile, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0666)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to create log file:", err)
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
		logMessage(LEVEL_FATAL, TYPE_NEORAY, "[PANIC!]", pmsg)
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
	if enableAsciiColorOutput {
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

func logMessage(lvl LogLevel, typ LogType, message ...interface{}) {
	if lvl < MINIMUM_LOG_LEVEL && verboseFile == nil {
		return
	}

	fatal := false
	colorCode := ""
	logLevelString := ""
	switch lvl {
	case LEVEL_DEBUG:
		logLevelString = "[DEBUG]"
		colorCode = AnsiWhite
	case LEVEL_TRACE:
		logLevelString = "[TRACE]"
		colorCode = AnsiGreen
	case LEVEL_WARN:
		logLevelString = "[WARNING]"
		colorCode = AnsiYellow
	case LEVEL_ERROR:
		logLevelString = "[ERROR]"
		colorCode = AnsiRed
	case LEVEL_FATAL:
		logLevelString = "[FATAL]"
		colorCode = AnsiRed
		fatal = true
	default:
		panic("invalid log level")
	}

	logTypeString := ""
	switch typ {
	case TYPE_NVIM:
		logTypeString = "[NVIM]"
	case TYPE_NEORAY:
		logTypeString = "[NEORAY]"
	case TYPE_RENDERER:
		logTypeString = "[RENDERER]"
	case TYPE_PERFORMANCE:
		logTypeString = "[PERFORMANCE]"
	default:
		panic("invalid log type")
	}

	logString := logLevelString + " " + logTypeString + " " + fmt.Sprintln(message...)

	// Print to stdout
	if enableAsciiColorOutput {
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

func logMessageFmt(level LogLevel, typ LogType, format string, args ...interface{}) {
	logMessage(level, typ, fmt.Sprintf(format, args...))
}

// Overload for built-in print function, use only for debugging
func print(msg ...interface{}) {
	assert_error(BUILD_TYPE != RELEASE, "print() function used in release build")
	fmt.Println(msg...)
}

func printf(format string, args ...interface{}) {
	assert_error(BUILD_TYPE != RELEASE, "printf() function used in release build")
	fmt.Printf(format, args...)
}

// This assert logs fatal when cond is false.
func assert(cond bool, message ...interface{}) {
	if cond == false {
		logMessage(LEVEL_FATAL, TYPE_NEORAY, "Assertion Failed:", fmt.Sprint(message...))
	}
}

// This assert logs error when cond is false.
func assert_error(cond bool, message ...interface{}) {
	if cond == false {
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Assertion Failed:", fmt.Sprint(message...))
	}
}
