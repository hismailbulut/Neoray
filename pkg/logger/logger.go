package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sqweek/dialog"
)

var guard sync.Mutex
var cache struct {
	initTime  time.Time // Time when logger was initialized
	name      string    // Name of the program using this logger
	version   Version   // Version of the program using this logger
	buildtype BuildType // Build type of the program using this logger
	file      *os.File  // File to write logs to
	color     bool      // Whether to use color in the output
}

func Init(name string, version Version, buildtype BuildType, color bool) {
	guard.Lock()
	defer guard.Unlock()
	cache.initTime = time.Now()
	cache.name = name
	cache.version = version
	cache.buildtype = buildtype
	cache.color = color
}

func timeString(t time.Time) string {
	return fmt.Sprintf("%s %d", t.UTC().Format("2006-01-02 15:04:05"), t.UnixMilli())
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
	cache.file, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0666)
	if err != nil {
		Log(ERROR, "Failed to create log file:", err)
		return
	}
	fmt.Fprintf(cache.file, "%s %s %s LOG %s\n", cache.name, cache.version, cache.buildtype, timeString(time.Now()))
}

// This function should be deferred after Init because it captures panics
func Shutdown() {
	// This will capture the panic and turns it to a fatal
	if pmsg := recover(); pmsg != nil {
		Log(FATAL, pmsg)
	} else {
		// Log(FATAL) already cleans up, we should only clean up if this is
		// a regular quit
		cleanup()
	}
}

// This function will always be called.
func cleanup() {
	guard.Lock()
	defer guard.Unlock()
	// If logfile is initialized then close it.
	if cache.file != nil {
		fmt.Fprintf(cache.file, "END OF LOG %s\n", timeString(time.Now()))
		cache.file.Close()
		cache.file = nil
	}
	// Reset terminal color
	if cache.color {
		fmt.Print(AnsiReset)
	}
}

func createCrashReport(msg string, isPanic bool) {
	guard.Lock()
	defer guard.Unlock()
	crash_file, err := os.Create("Neoray_crash.log")
	if err == nil {
		defer crash_file.Close()
		fmt.Fprintf(crash_file, "%s %s Crash Report\n", cache.name, cache.version)
		fmt.Fprintf(crash_file, "Build Type: %s\n", cache.buildtype)
		fmt.Fprintf(crash_file, "Start Time: %s\n", timeString(cache.initTime))
		fmt.Fprintf(crash_file, "Crash Time: %s\n", timeString(time.Now()))
		fmt.Fprintf(crash_file, "\n%s %s\n", cache.name, "is crashed because of the following reason:")
		fmt.Fprintf(crash_file, "%s\n", msg)
		// Write stack trace
		fmt.Fprintf(crash_file, "\n%s\n", "Stack trace:")
		stackTrace := make([]byte, 1<<15) // This is highly enough
		stackLen := runtime.Stack(stackTrace, true)
		crash_file.Write(stackTrace[:stackLen])
	}
}

func Log(logLevel LogLevel, message ...any) {
	guard.Lock()
	buildtype := cache.buildtype
	guard.Unlock()

	if buildtype == ReleaseBuild && logLevel < TRACE {
		return
	}

	messageStr := strings.TrimRight(fmt.Sprintln(message...), "\n\t ")

	logString := fmt.Sprintf("%s %s", logLevel, messageStr)

	// Print to stdout
	if cache.color {
		fmt.Printf("%s%s\n", string(logLevel.Color()), logString)
	} else {
		fmt.Printf("%s\n", logString)
	}

	// Print to verbose file if opened
	guard.Lock()
	if cache.file != nil {
		fmt.Fprintf(cache.file, "%s\n", logString)
	}
	guard.Unlock()

	if logLevel == FATAL {
		// Create crash report file
		createCrashReport(logString, false)
		// Show error dialog
		dialog.Message(logString).Error()
		// Cleanup and shutdown.
		cleanup()
		os.Exit(1)
	}
}

func LogF(level LogLevel, format string, args ...any) {
	Log(level, fmt.Sprintf(format, args...))
}
