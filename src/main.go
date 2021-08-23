package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"
)

const (
	DEBUG   = 0
	RELEASE = 1

	TITLE         = "neoray"
	VERSION_MAJOR = 0
	VERSION_MINOR = 0
	VERSION_PATCH = 7
	WEBPAGE       = "github.com/hismailbulut/neoray"
	LICENSE       = "MIT"
)

func init() {
	// Opengl and glfw needs this.
	runtime.LockOSThread()
	// Enabling this helps us to catch and print segfaults.
	debug.SetPanicOnFault(true)
}

// singleton is main instance of the editor and there can be only one
// editor in this program. singleton is not threadsafe and can not be
// accessed at same time from different threads or goroutines. Most of the
// functions accessing it and these functions also not thread safe.
var singleton Editor

// Given arguments when starting this editor.
var editorParsedArgs ParsedArgs

func main() {
	start := time.Now()
	// If --verbose flag is set then new file will be created with given name
	// and we need to close this file. This function will check if the file is open
	// and than closes it. Also recovers panic and prints to the logfile if the program panics.
	// Only main goroutine panic can be captured.
	defer shutdownLogger()
	// Trackers are debug functions and collects data about what function
	// called how many times and it's execution time. Works in only debug build
	// You need to defer measure_execution_time()() in the beginning of the function
	// you want to track.
	init_function_time_tracker()
	defer close_function_time_tracker()
	// Parse args
	editorParsedArgs = ParseArgs(os.Args[1:])
	// If ProcessBefore returns true, neoray will not start.
	if editorParsedArgs.ProcessBefore() {
		return
	}
	// Starts a pprof server. This function is only implemented in debug build.
	start_pprof()
	// Initializing editor will initialize everything.
	singleton.Initialize()
	// And shutdown will frees resources and closes everything.
	defer singleton.Shutdown()
	// Some arguments must be processed after initializing.
	editorParsedArgs.ProcessAfter()
	// Start time information
	logMessage(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Start time:", time.Since(start))
	// MainLoop is main loop of the neoray.
	singleton.MainLoop()
}

func isDebugBuild() bool {
	return BUILD_TYPE == DEBUG
}

func buildTypeString() string {
	if isDebugBuild() {
		return "Debug"
	}
	return "Release"
}

func versionString() string {
	return fmt.Sprintf("v%d.%d.%d", VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH)
}
