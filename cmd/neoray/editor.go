package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

type Editor struct {
	// Neovim child process, nvim_process.go
	nvim NvimProcess
	// Window, window.go
	window Window
	// Grid is a neovim's windows if multigrid option enabled, otherwise full window.
	// Grid has cells and it's attributes, and all information how cells and fonts will be rendered.
	// grid.go,
	// TODO: support multigrid
	grid Grid
	// Cursor represents a neovim cursor and all it's information, cursor.go
	cursor Cursor
	// Mode is current nvim mode information struct (normal, visual etc.)
	// We need for cursor rendering and some other stuff, more may future
	mode Mode
	// Renderer is a struct that holds sdl and ttf rendering information,
	// and has direct rendering abilities.
	// NOTE: renderer is very slow
	renderer Renderer
	// UIOptions is a struct, holds some user ui options like guifont.
	options UIOptions
	// Quit requested is a boolean, if it is true the program will be shutdown at begin of the next loop
	quit_requested bool
}

const (
	NEORAY_NAME          = "Neoray"
	NEORAY_VERSION_MAJOR = 0
	NEORAY_VERSION_MINOR = 0
	NEORAY_VERSION_PATCH = 1
	NEORAY_WEBPAGE       = "github.com/hismailbulut/Neoray"
	NEORAY_LICENSE       = "GPLv3"
)

const TARGET_TPS = 60

// temporary
const FONT_NAME = "Consolas"
const FONT_SIZE = 17
const BG_TRANSPARENCY = 255

func (editor *Editor) Initialize() {
	// pprof
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// We will first initialize neovim process, this function is
	// concurrent. We can set and initialize our program when waiting neovim.
	editor.nvim = CreateNvimProcess()

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS); err != nil {
		log.Fatalln(err)
	}
	editor.window = CreateWindow(1024, 768, NEORAY_NAME)

	editor.grid = CreateGrid()

	editor.cursor = Cursor{}

	editor.mode = Mode{
		mode_infos: make(map[string]ModeInfo),
	}

	editor.renderer = CreateRenderer(
		&editor.window, CreateFont(FONT_NAME, FONT_SIZE))

	editor.options = UIOptions{}

	// We initialized everything we need, now this is the time for
	// connecting UI
	editor.nvim.StartUI(editor)
}

func (editor *Editor) MainLoop() {
	ticker := time.NewTicker(time.Millisecond * (1000 / TARGET_TPS))
	defer ticker.Stop()
	for !editor.quit_requested {
		HandleSDLEvents(editor)
		editor.window.Update(editor)
		<-ticker.C
	}
}

func (editor *Editor) Shutdown() {
	editor.nvim.Close()
	editor.window.Close()
	editor.renderer.Close()
	sdl.Quit()
}
