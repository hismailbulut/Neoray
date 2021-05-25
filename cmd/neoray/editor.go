package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

// These are the global variables of neoray. They are same in everywhere.
// Some of them are initialized at runtime. Therefore we must be carefull when we
// use them. If you add more here just write some information about it.
var (
	// Initialized in CreateRenderer
	GLOB_CellWidth    = 0
	GLOB_CellHeight   = 0
	GLOB_RowCount     = 0
	GLOB_ColumnCount  = 0
	GLOB_WindowWidth  = 800
	GLOB_WindowHeight = 600
	// MainLoop is setting these
	GLOB_FramesPerSecond         = 0
	GLOB_DeltaTime       float32 = 0.16
)

type Editor struct {
	// Neovim child process, nvim_process.go
	nvim NvimProcess
	// Window, window.go
	window Window
	// Grid is a neovim window if multigrid option is enabled, otherwise full screen.
	// Grid has cells and it's attributes, and contains all information how cells and fonts will be rendered.
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

// Consolas
// Ubuntu Mono
// GoMono
// Cousine
// Hack
// JetBrains Mono
// Caskadyia Cove
const FONT_NAME = "Go Mono"
const FONT_SIZE = 14
const TARGET_TPS = 60

func (editor *Editor) Initialize() {
	// NOTE: disable on release build
	startupTime := time.Now()
	init_function_time_tracker()
	// pprof for debugging
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create pprof server.")
		}
	}()

	editor.nvim = CreateNvimProcess()

	if err := sdl.Init(sdl.INIT_EVENTS); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL2:", err)
	}

	editor.window = CreateWindow(GLOB_WindowWidth, GLOB_WindowHeight, NEORAY_NAME)

	editor.grid = CreateGrid()

	editor.cursor = Cursor{}

	editor.mode = CreateMode()

	editor.renderer = CreateRenderer(
		&editor.window, CreateFont(FONT_NAME, FONT_SIZE))

	editor.options = UIOptions{}

	editor.quit_requested_chan = make(chan bool)

	editor.nvim.StartUI(editor)

	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE, "Startup time:", time.Since(startupTime))
}

func (editor *Editor) MainLoop() {
	mainLoopBeginTime := time.Now()
	ticker := time.NewTicker(time.Millisecond * (1000 / TARGET_TPS))
	defer ticker.Stop()
	fpsTimer := time.Now()
	fps := 0
	for !editor.quit_requested {
		select {
		case <-editor.quit_requested_chan:
			editor.quit_requested = true
			continue
		default:
		}
		HandleSDLEvents(editor)
		editor.window.Update(editor)
		fps++
		if time.Since(fpsTimer) > time.Second {
			GLOB_FramesPerSecond = fps
			GLOB_DeltaTime = float32(fps) / 1000
			fps = 0
			fpsTimer = time.Now()
		}
		<-ticker.C
	}
	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE, "Total execution time:", time.Since(mainLoopBeginTime))
}

func (editor *Editor) Shutdown() {
	editor.nvim.Close()
	editor.window.Close()
	editor.renderer.Close()
	sdl.Quit()
	close_function_time_tracker()
}
