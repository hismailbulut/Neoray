package main

import (
	"os"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/logger"
)

const (
	NAME          = "Neoray"
	VERSION_MAJOR = 0
	VERSION_MINOR = 2
	VERSION_PATCH = 1
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
	var err error
	var quit bool
	Editor.parsedArgs, err, quit = ParseArgs(os.Args[1:])
	if err != nil {
		logger.Log(logger.FATAL, err)
	}
	if quit {
		return
	}
	// If ProcessBefore returns true, neoray will not start.
	// Initializes logfile if required argument passed
	// And also initializes server if required argument passed
	quit = Editor.parsedArgs.ProcessBefore()
	if quit {
		return
	}
	// Initializing editor will initialize everything
	InitEditor()
	// And shutdown will frees resources and closes everything
	defer ShutdownEditor()
	// Some arguments must be processed after initialization
	Editor.parsedArgs.ProcessAfter()
	// Start time information
	logger.Log(logger.TRACE, "Initialization time:", time.Since(StartTime))
	// MainLoop is main loop of the neoray.
	MainLoop()
}
