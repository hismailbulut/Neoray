package main

import (
	"log"
	"os"
)

const (
	BUILD_TYPE_DEBUG   = 0
	BUILD_TYPE_RELEASE = 1
	NEORAY_BUILD_TYPE  = BUILD_TYPE_DEBUG

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
	log.SetFlags(0)
}

// EditorSingleton is main instance of the editor and there can be only one
// editor in this program. EditorSingleton is not threadsafe and can not be
// accessed at same time from different threads or goroutines. Most of the
// functions accessing it and these functions also not thread safe.
var EditorSingleton Editor

func main() {
	// Set some build type specific options.
	switch NEORAY_BUILD_TYPE {
	case BUILD_TYPE_DEBUG:
		start_pprof()
		init_function_time_tracker()
		defer close_function_time_tracker()
		MINIMUM_LOG_LEVEL = LOG_LEVEL_DEBUG
	case BUILD_TYPE_RELEASE:
		MINIMUM_LOG_LEVEL = LOG_LEVEL_WARN
	default:
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Unknown build type.")
	}

	argOptions, nvimArgs := ParseArgs(os.Args[1:])
	// If parse before returns true, we will not start neoray.
	if argOptions.ProcessBefore() {
		return
	}

	EditorSingleton = Editor{}
	// Initializing editor is initializes everything.
	EditorSingleton.Initialize(nvimArgs)
	// And shutdown will frees resources and closes neovim.
	defer EditorSingleton.Shutdown()
	// Some arguments must be processed after initializing.
	argOptions.ProcessAfter()
	// MainLoop is main loop of the neoray.
	EditorSingleton.MainLoop()
}

func isDebugBuild() bool {
	return NEORAY_BUILD_TYPE == BUILD_TYPE_DEBUG
}
