package main

import (
	"time"

	"github.com/veandco/go-sdl2/sdl"
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
	// These are the global variables of neoray. They are same in everywhere.
	// Some of them are initialized at runtime. Therefore we must be carefull when we
	// use them. If you add more here just write some information about it.
	// Initializing in CreateRenderer
	cellWidth  int
	cellHeight int
	// Initializing in CalculateCellCount
	rowCount    int
	columnCount int
	cellCount   int
	// Initializing in Editor.MainLoop
	framesPerSecond int
	deltaTime       float32
}

// Consolas
// Ubuntu Mono
// GoMono
// Cousine
// Hack
// JetBrains Mono
// Caskadyia Cove
const FONT_NAME = "Hack"
const FONT_SIZE = 14
const TARGET_TPS = 60

func (editor *Editor) Initialize() {
	startupTime := time.Now()

	init_function_time_tracker()

	editor.nvim = CreateNvimProcess()

	if err := sdl.Init(sdl.INIT_EVENTS); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL2:", err)
	}

	editor.window = CreateWindow(800, 600, NEORAY_NAME)

	editor.grid = CreateGrid()

	editor.cursor = Cursor{}

	editor.mode = CreateMode()

	editor.renderer = CreateRenderer(CreateFont(FONT_NAME, FONT_SIZE))

	editor.options = UIOptions{}

	editor.quit_requested_chan = make(chan bool)

	editor.nvim.StartUI()

	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE, "Startup time:", time.Since(startupTime))
}

func (editor *Editor) MainLoop() {
	programBegin := time.Now()
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
		HandleSDLEvents()
		HandleNvimRedrawEvents()
		editor.window.Update()
		editor.cursor.Update()
		fps++
		if time.Since(fpsTimer) > time.Second {
			editor.framesPerSecond = fps
			editor.deltaTime = float32(fps) / 1000
			fps = 0
			fpsTimer = time.Now()
		}
		<-ticker.C
	}
	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE,
		"Program finished. Total execution time:", time.Since(programBegin))
}

func (editor *Editor) Shutdown() {
	editor.nvim.Close()
	editor.grid.Destroy()
	editor.window.Close()
	editor.renderer.Close()
	sdl.Quit()
	close_function_time_tracker()
}
