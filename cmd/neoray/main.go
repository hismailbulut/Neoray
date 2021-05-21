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

func main() {
	editor := Editor{}

	editor.Initialize()
	defer editor.Shutdown()

	editor.MainLoop()
}
