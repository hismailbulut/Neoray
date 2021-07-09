package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"
)

const (
	DEBUG   = 0
	RELEASE = 1

	TITLE         = "neoray"
	VERSION_MAJOR = 0
	VERSION_MINOR = 0
	VERSION_PATCH = 5
	WEBPAGE       = "github.com/hismailbulut/Neoray"
	LICENSE       = "MIT"
)

// NOTE: This source code is documented by me and I don't know English well.
// Sorry about the typos and expression disorders. If you find any of these please
// correct me with a pr or something else you can communicate with me.

func init() {
	runtime.LockOSThread()
	log.SetFlags(0)
}

// EditorSingleton is main instance of the editor and there can be only one
// editor in this program. EditorSingleton is not threadsafe and can not be
// accessed at same time from different threads or goroutines. Most of the
// functions accessing it and these functions also not thread safe.
var EditorSingleton Editor

// Given arguments when starting this editor.
var EditorParsedArgs ParsedArgs

func main() {
	start := time.Now()
	// If --verbose flag is set then new file will be created with given name
	// and we need to close this file. This function will check if the file is open
	// and than closes it. Also recovers panic and prints to the logfile if the program panics.
	// Only main goroutine panic can be captured.
	defer close_logger()
	// Trackers are debug functions and collects data about what function
	// called how many times and it's execution time. Works in only debug build
	init_function_time_tracker()
	defer close_function_time_tracker()
	// Parse args
	EditorParsedArgs = ParseArgs(os.Args[1:])
	// If ProcessBefore returns true, neoray will not start.
	if EditorParsedArgs.ProcessBefore() {
		return
	}
	// Starts a pprof server. This function is only implemented in debug build.
	start_pprof()
	// Initializing editor will initialize everything.
	EditorSingleton.Initialize()
	// And shutdown will frees resources and closes everything.
	defer EditorSingleton.Shutdown()
	// Some arguments must be processed after initializing.
	EditorParsedArgs.ProcessAfter()
	// Start time information
	log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Start time:", time.Since(start))
	// MainLoop is main loop of the neoray.
	EditorSingleton.MainLoop()
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
	return fmt.Sprintf("%d.%d.%d", VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH)
}
