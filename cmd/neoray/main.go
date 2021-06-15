package main

import (
	"log"
	"os"
	"runtime"
)

const (
	DEBUG   = 0
	RELEASE = 1

	NEORAY_NAME          = "Neoray"
	NEORAY_VERSION_MAJOR = 0
	NEORAY_VERSION_MINOR = 0
	NEORAY_VERSION_PATCH = 3
	NEORAY_WEBPAGE       = "github.com/hismailbulut/Neoray"
	NEORAY_LICENSE       = "GPLv3"
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
var EditorArgs Args

func main() {
	init_function_time_tracker()
	defer close_function_time_tracker()
	// Parse args
	EditorArgs = ParseArgs(os.Args[1:])
	// If parse before returns true, we will not start neoray.
	if EditorArgs.ProcessBefore() {
		return
	}
	start_pprof()
	EditorSingleton = Editor{}
	// Initializing editor is initializes everything.
	EditorSingleton.Initialize()
	// And shutdown will frees resources and closes neovim.
	defer EditorSingleton.Shutdown()
	// Some arguments must be processed after initializing.
	EditorArgs.ProcessAfter()
	// MainLoop is main loop of the neoray.
	EditorSingleton.MainLoop()
}

func isDebugBuild() bool {
	return BUILD_TYPE == DEBUG
}

func getBuildTypeString() string {
	if BUILD_TYPE == DEBUG {
		return "Debug"
	}
	return "Release"
}
