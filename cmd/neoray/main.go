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
	NEORAY_VERSION_PATCH = 1
	NEORAY_WEBPAGE       = "github.com/hismailbulut/Neoray"
	NEORAY_LICENSE       = "GPLv3"
)

// NOTE: This source code is documented by me and I don't know English well.
// Sorry about the typos and expression disorders. If you find any of these please
// correct me with a pr or something else you can communicate with me.

func init() {
	log.SetFlags(0)
}

func main() {
	// NOTE: Disable on release build
	start_pprof()
	editor := Editor{}
	// Initializing editor is initializes everything.
	editor.Initialize()
	// And shutdown will frees resources and closes neovim.
	defer editor.Shutdown()
	// MainLoop is main loop of the neoray.
	editor.MainLoop()
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
