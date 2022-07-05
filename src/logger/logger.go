package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/sqweek/dialog"
)

var guard sync.Mutex
var cache struct {
	name      string
	version   Version
	buildtype BuildType
	file      *os.File
	color     bool
}

func Init(name string, version Version, buildtype BuildType, color bool) {
	guard.Lock()
	defer guard.Unlock()
	cache.name = name
	cache.version = version
	cache.buildtype = buildtype
	cache.color = color
}

// Defer this
func Shutdown() {
	// If we are panicking print it to logfile.
	// Also the stack trace will be printed after
	// fatal error.
	if pmsg := recover(); pmsg != nil {
		// Create crash report.
		Log(FATAL, pmsg)
	}
	cleanup()
}

func InitFile(filename string) {
	guard.Lock()
	defer guard.Unlock()
	if cache.file != nil {
		return
	}
	path, err := filepath.Abs(filename)
	if err != nil {
		Log(ERROR, "Failed to get absolute path:", err)
		return
	}
	cache.file, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0666)
	if err != nil {
		Log(ERROR, "Failed to create log file:", err)
		return
	}
	// Print informations to log file.
	cache.file.WriteString(fmt.Sprintln("\n", cache.name, cache.version, cache.buildtype, "LOG", time.Now().UTC()))
}

// This function will always be called.
func cleanup() {
	guard.Lock()
	defer guard.Unlock()
	// If logfile is initialized then close it.
	if cache.file != nil {
		cache.file.Close()
	}
	// Reset terminal color
	if cache.color {
		fmt.Print(AnsiReset)
	}
}

func createCrashReport(msg string) {
	guard.Lock()
	defer guard.Unlock()
	crash_file, err := os.Create("Neoray_crash.log")
	if err == nil {
		defer crash_file.Close()
		crash_file.WriteString(fmt.Sprintln(cache.name, cache.version, cache.buildtype, "Crash Report", time.Now().UTC()))
		crash_file.WriteString("Please open an issue in github with this file.\n")
		crash_file.WriteString("The program is crashed because of the following reasons:\n")
		crash_file.WriteString(msg)
		crash_file.WriteString("\ngoroutine dump:\n")
		stackTrace := make([]byte, 1<<15)
		stackLen := runtime.Stack(stackTrace, true)
		crash_file.WriteString(string(stackTrace[:stackLen]))
	}
}

func Log(logLevel LogLevel, message ...any) {
	guard.Lock()
	buildtype := cache.buildtype
	guard.Unlock()

	if buildtype == ReleaseBuild && logLevel < TRACE {
		return
	}

	logString := logLevel.String() + " " + fmt.Sprintln(message...)

	// Print to stdout
	if cache.color {
		fmt.Print(string(logLevel.Color()) + logString)
	} else {
		fmt.Print(logString)
	}

	// Print to verbose file if opened
	guard.Lock()
	if cache.file != nil {
		cache.file.WriteString(logString)
	}
	guard.Unlock()

	if logLevel == FATAL {
		// Create crash report file
		createCrashReport(logString)
		// Open message box with error
		dialog.Message(logString).Error()
		// Cleanup and shutdown.
		cleanup()
		os.Exit(1)
	}
}

func LogF(level LogLevel, format string, args ...any) {
	Log(level, fmt.Sprintf(format, args...))
}
