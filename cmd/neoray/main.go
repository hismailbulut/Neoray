package main

import (
	"log"
	"os"
)

const (
	DEBUG_BUILD   = 0
	RELEASE_BUILD = 1

	NEORAY_NAME          = "Neoray"
	NEORAY_VERSION_MAJOR = 0
	NEORAY_VERSION_MINOR = 0
	NEORAY_VERSION_PATCH = 2
	NEORAY_WEBPAGE       = "github.com/hismailbulut/Neoray"
	NEORAY_LICENSE       = "GPLv3"

	NEORAY_BUILD_TYPE = DEBUG_BUILD
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

var NeovimArgs []string

func preprocessArgs() bool {
	dontStart := false
	for _, arg := range os.Args[1:] {
		if beginsWith(arg, "--single-instance=", "-si=") {
			fileName := afterSubstr(arg, "--single-instance=", "-si=")
			if fileName != "" && SendOpenFile(fileName) {
				dontStart = true
			} else {
				CreateServer()
			}
		} else {
			NeovimArgs = append(NeovimArgs, arg)
		}
	}
	return dontStart
}

func main() {
	switch NEORAY_BUILD_TYPE {
	case DEBUG_BUILD:
		start_pprof()
		init_function_time_tracker()
		defer close_function_time_tracker()
		MINIMUM_LOG_LEVEL = LOG_LEVEL_DEBUG
	case RELEASE_BUILD:
		MINIMUM_LOG_LEVEL = LOG_LEVEL_WARN
	default:
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Unknown build type.")
	}

	if preprocessArgs() {
		return
	}

	EditorSingleton = Editor{}
	// Initializing editor is initializes everything.
	EditorSingleton.Initialize()
	// And shutdown will frees resources and closes neovim.
	defer EditorSingleton.Shutdown()
	// MainLoop is main loop of the neoray.
	EditorSingleton.MainLoop()
}
