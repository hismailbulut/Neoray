package main

import (
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
	quit_requested      bool
	quit_requested_chan chan bool
}

// temporary
// NOTE: This options must get from user settings.
const TARGET_TPS = 60
const WINDOW_WIDTH = 800
const WINDOW_HEIGHT = 600
const FONT_NAME = "Cousine"
const FONT_SIZE = 17
const BG_TRANSPARENCY = 245

func (editor *Editor) Initialize() {
	// pprof for debugging
	// NOTE: disable on release build
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create pprof server.")
		}
	}()

	editor.nvim = CreateNvimProcess()

	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_EVENTS); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL2:", err)
	}
	editor.window = CreateWindow(WINDOW_WIDTH, WINDOW_HEIGHT, NEORAY_NAME)

	editor.grid = CreateGrid()

	editor.cursor = Cursor{}

	editor.mode = Mode{
		mode_infos: make(map[string]ModeInfo),
	}

	editor.renderer = CreateRenderer(
		&editor.window, CreateFont(FONT_NAME, FONT_SIZE))

	editor.options = UIOptions{}

	// We initialized everything, we can connect UI now.
	editor.nvim.StartUI(editor)

	editor.quit_requested_chan = make(chan bool)
}

func (editor *Editor) MainLoop() {
	ticker := time.NewTicker(time.Millisecond * (1000 / TARGET_TPS))
	defer ticker.Stop()
	for !editor.quit_requested {
		select {
		case <-editor.quit_requested_chan:
			editor.quit_requested = true
			continue
		default:
		}
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
