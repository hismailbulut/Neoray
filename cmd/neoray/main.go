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
	VERSION_PATCH = 7
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
	// Initialize logger
	logger.Init(NAME, logger.Version{Major: VERSION_MAJOR, Minor: VERSION_MINOR, Patch: VERSION_PATCH}, bench.BUILD_TYPE, true)
	defer logger.Shutdown()
	// Print benchmark results
	defer bench.PrintResults(os.Stdout)
	// Create global editor singleton
	editor := NewEditor()
	// Parse args
	// If ProcessBefore returns true, neoray will not start.
	quit := editor.ProcessArgsBeforeInit()
	if quit {
		return
	}
	// Enable high resolution timer on windows for high refresh rate, improves animation quality
	BeginHighresTimer()
	defer EndHighresTimer()
	// Initializing editor will initialize everything
	err := editor.Init()
	if err != nil {
		logger.Fatal("Failed to initialize editor:", err)
	}
	// And shutdown will frees resources and closes everything
	defer editor.Terminate()
	// Some arguments must be processed after initialization
	editor.ProcessArgsAfterInit()
	// Start time information
	logger.Trace("Initialization time:", time.Since(StartTime))
	// MainLoop is main loop of the neoray.
	editor.MainLoop()
}
