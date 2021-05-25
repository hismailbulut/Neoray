package main

import (
	"log"
)

const (
	NEORAY_NAME          = "Neoray"
	NEORAY_VERSION_MAJOR = 0
	NEORAY_VERSION_MINOR = 0
	NEORAY_VERSION_PATCH = 1
	NEORAY_WEBPAGE       = "github.com/hismailbulut/Neoray"
	NEORAY_LICENSE       = "GPLv3"
)

func init() {
	log.SetFlags(0)
}

// NOTE: This source code is documented by me and I don't know English well.
// Sorry about the typos and expression disorders. If you find any of these please
// correct me with a pr or something else you can communicate with me.

func main() {
	editor := Editor{}
	// Initializing editor is initializes everything.
	editor.Initialize()
	// And shutdown will frees resources and closes neovim.
	defer editor.Shutdown()
	// MainLoop is main loop of the neoray.
	editor.MainLoop()
}
