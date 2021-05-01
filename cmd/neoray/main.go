package main

import (
	"runtime"
	"time"

	"github.com/chunqian/go-raylib/raylib"
)

const (
	NEORAY_NAME          = "Neoray"
	NEORAY_VERSION_MAJOR = 0
	NEORAY_VERSION_MINOR = 0
	NEORAY_VERSION_PATCH = 1
	NEORAY_WEBPAGE       = "github.com/hismailbulut/Neoray"
	NEORAY_LICENSE       = "GPLv3"
)

// temporary
const FONT_NAME = "UbuntuMono"
const FONT_SIZE = 18
const TRANSPARENCY = 245

const TARGET_FPS = 120

type Editor struct {
	nvim   NvimProcess
	window Window
}

func (editor *Editor) shutdown() {
	editor.nvim.Close()
	editor.window.Close()
}

func init() {
	runtime.LockOSThread()
}

func main() {
	editor := Editor{
		nvim:   CreateNvimProcess(),
		window: CreateWindow(1024, 768, NEORAY_NAME, FONT_NAME, FONT_SIZE),
	}
	defer editor.shutdown()

	editor.nvim.StartUI(&editor.window)

	ticker := time.NewTicker(time.Millisecond * (1000 / TARGET_FPS))
	defer ticker.Stop()

	for !raylib.WindowShouldClose() {
		editor.window.Update(&editor.nvim)
		editor.window.Render()
		<-ticker.C
	}
}
