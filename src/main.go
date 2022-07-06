package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"time"

	_ "github.com/hismailbulut/neoray/src/assets"
	"github.com/hismailbulut/neoray/src/bench"
	"github.com/hismailbulut/neoray/src/logger"
)

const (
	NAME          = "Neoray"
	VERSION_MAJOR = 0
	VERSION_MINOR = 2
	VERSION_PATCH = 0
	WEBPAGE       = "github.com/hismailbulut/Neoray"
	LICENSE       = "MIT"
)

// Start time of the program
var StartTime time.Time

func init() {
	runtime.LockOSThread()
	// Enabling this helps us to catch and print segfaults (Does it?)
	debug.SetPanicOnFault(true)
}

func main() {
	StartTime = time.Now()
	// Init logger
	logger.Init(NAME, logger.Version{Major: VERSION_MAJOR, Minor: VERSION_MINOR, Patch: VERSION_PATCH}, bench.BUILD_TYPE, true)
	defer logger.Shutdown()
	// Print benchmark results
	defer bench.PrintResults()
	// Parse args
	Editor.parsedArgs = ParseArgs(os.Args[1:])
	// If ProcessBefore returns true, neoray will not start.
	// Initializes logfile if required argument passed
	// And also initializes server if required argument passed
	if Editor.parsedArgs.ProcessBefore() {
		return
	}
	// Initializing editor will initialize everything
	InitEditor()
	// And shutdown will frees resources and closes everything
	defer ShutdownEditor()
	// Some arguments must be processed after initialization
	Editor.parsedArgs.ProcessAfter()
	// Start time information
	logger.Log(logger.TRACE, "Start time:", time.Since(StartTime))
	// MainLoop is main loop of the neoray.
	MainLoop()
}

// This assert logs fatal when cond is false.
func assert(cond bool, message ...any) {
	if cond == false {
		logger.Log(logger.FATAL, "Assertion Failed:", fmt.Sprint(message...))
	}
}

// This assert logs error when cond is false.
func assert_error(cond bool, message ...any) {
	if cond == false {
		logger.Log(logger.ERROR, "Assertion Failed:", fmt.Sprint(message...))
	}
}
