package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

const (
	NEORAY_NAME          = "Neoray"
	NEORAY_VERSION_MAJOR = 0
	NEORAY_VERSION_MINOR = 0
	NEORAY_VERSION_PATCH = 2
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
// accessed at same time from different threads or goroutines.
var EditorSingleton Editor

func main() {
	// NOTE: Disable on release build
	start_pprof()

	EditorSingleton = Editor{}
	// Initializing editor is initializes everything.
	EditorSingleton.Initialize()
	// And shutdown will frees resources and closes neovim.
	defer EditorSingleton.Shutdown()
	// MainLoop is main loop of the neoray.
	EditorSingleton.MainLoop()
}

func start_pprof() {
	// pprof for debugging
	// NOTE: disable on release build
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create pprof server.")
		}
	}()
}
