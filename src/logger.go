package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sqweek/dialog"
)

type AnsiTermColor string

const enableAsciiColorOutput = true

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

func (logLevel LogLevel) String() string {
	switch logLevel {
	case LEVEL_DEBUG:
		return "[DEBUG]"
	case LEVEL_TRACE:
		return "[TRACE]"
	case LEVEL_WARN:
		return "[WARNING]"
	case LEVEL_ERROR:
		return "[ERROR]"
	case LEVEL_FATAL:
		return "[FATAL]"
	default:
		panic("invalid log level")
	}
}

func getColorOfLogLevel(logLevel LogLevel) AnsiTermColor {
	switch logLevel {
	case LEVEL_DEBUG:
		return AnsiWhite
	case LEVEL_TRACE:
		return AnsiGreen
	case LEVEL_WARN:
		return AnsiYellow
	case LEVEL_ERROR:
		return AnsiRed
	case LEVEL_FATAL:
		return AnsiRed
	default:
		panic("invalid log level")
	}
}

const (
	// Log types, makes easy to understand where the message coming from
	TYPE_NVIM LogType = iota
	TYPE_NEORAY
	TYPE_RENDERER
	TYPE_PERFORMANCE
	TYPE_NETWORK
)

func (logType LogType) String() string {
	switch logType {
	case TYPE_NVIM:
		return "[NVIM]"
	case TYPE_NEORAY:
		return "[NEORAY]"
	case TYPE_RENDERER:
		return "[RENDERER]"
	case TYPE_PERFORMANCE:
		return "[PERFORMANCE]"
	case TYPE_NETWORK:
		return "[NETWORK]"
	default:
		panic("invalid log type")
	}
}

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

func logMessage(logLevel LogLevel, logType LogType, message ...interface{}) {
	if logLevel < MINIMUM_LOG_LEVEL && verboseFile == nil {
		return
	}

	colorCode := getColorOfLogLevel(logLevel)

	logString := logLevel.String() + " " + logType.String() + " " + fmt.Sprintln(message...)

	// Print to stdout
	if enableAsciiColorOutput {
		fmt.Print(string(colorCode) + logString)
	} else {
		fmt.Print(logString)
	}

	// Print to verbose file if opened
	if verboseFile != nil {
		verboseFile.WriteString(logString)
	}

	if logLevel == LEVEL_FATAL {
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

// NOTE: print* functions only used for debugging purposes. Use logMessage* for
// actually logging them and never build a release version has print* function
// usage.

// Overload for built-in print function, use only for debugging
func print(msg ...interface{}) {
	assert_error(BUILD_TYPE != RELEASE, "print() function used in release build")
	fmt.Println(msg...)
}

// Colored print
func printc(color AnsiTermColor, msg ...interface{}) {
	assert_error(BUILD_TYPE != RELEASE, "printc() function used in release build")
	fmt.Print(color)
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
