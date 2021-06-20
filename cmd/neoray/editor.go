package main

import (
	"math"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	MINIMUM_FONT_SIZE    = 7
	DEFAULT_FONT_SIZE    = 12
	DEFAULT_TRANSPARENCY = 1
	DEFAULT_TARGET_TPS   = 60
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
	// PopupMenu is the only popup menu in this program for right click menu.
	popupMenu PopupMenu
	// If quitRequested is true the program will quit.
	quitRequestedChan chan bool
	// This is for resizing from nvim side, first we will send resize request to neovim,
	// than we wait for the resize call from neovim side. When waiting this we dont want
	// to render because its dangerous.
	waitingResize bool
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
	// Transparency of window background min 0, max 1
	framebufferTransparency float32
	// Target ticks per second
	targetTPS int
	// Server for singleinstance
	server *TCPServer
}

func (editor *Editor) Initialize() {
	startupTime := time.Now()

	editor.initDefaults()
	editor.nvim = CreateNvimProcess()

	if err := glfw.Init(); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize glfw:", err)
	}
	editor.window = CreateWindow(800, 600, TITLE)
	InitializeInputEvents()

	editor.grid = CreateGrid()
	editor.mode = CreateMode()
	editor.cursor = CreateCursor()
	editor.options = UIOptions{}

	editor.renderer = CreateRenderer()
	editor.popupMenu = CreatePopupMenu()

	editor.quitRequestedChan = make(chan bool)
	editor.nvim.StartUI()

	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE, "Startup time:", time.Since(startupTime))
}

func (editor *Editor) initDefaults() {
	editor.framebufferTransparency = DEFAULT_TRANSPARENCY
	editor.targetTPS = DEFAULT_TARGET_TPS
}

func (editor *Editor) MainLoop() {
	// For measuring total time of the program.
	programBegin := time.Now()
	// Ticker's interval
	interval := time.Second / time.Duration(editor.targetTPS)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// For measuring tps.
	var elapsed float32
	ticks := 0
	// For measuring delta time
	loopBegin := time.Now()
	// For secure quit.
	quitRequestedFromNvim := false
	// Mainloop
MAINLOOP:
	for !editor.window.handle.ShouldClose() {
		select {
		case ticktime := <-ticker.C:
			// Calculate delta time
			editor.deltaTime = float32(ticktime.Sub(loopBegin)) / float32(time.Second)
			loopBegin = ticktime
			elapsed += editor.deltaTime
			ticks++
			// Calculate ticks per second
			if elapsed >= 1 {
				editor.framesPerSecond = ticks
				ticks = 0
				elapsed = 0
			}
			// Update program. Order is important!
			if editor.server != nil {
				editor.server.Process()
			}
			HandleNvimRedrawEvents()
			if !editor.waitingResize {
				editor.window.Update()
				editor.cursor.Update()
				editor.renderer.Update()
			}
			glfw.PollEvents()
		case <-editor.quitRequestedChan:
			editor.window.handle.SetShouldClose(true)
			quitRequestedFromNvim = true
		}
	}
	if !quitRequestedFromNvim {
		// Maybe there is a unsaved files in neovim. Instead of immediately closing
		// we will send simple quit command to neovim and if there is a unsaved files
		// the neovim will handle it and user will not lose its progress.
		editor.window.handle.SetShouldClose(false)
		go editor.nvim.ExecuteVimScript(":qa")
		goto MAINLOOP
	}
	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE, "Program finished. Total execution time:", time.Since(programBegin))
}

func (editor *Editor) calculateCellCount() {
	editor.columnCount = editor.window.width / editor.cellWidth
	editor.rowCount = editor.window.height / editor.cellHeight
	editor.cellCount = editor.columnCount * editor.rowCount
}

func (editor *Editor) backgroundAlpha() uint8 {
	transparency := math.Min(1, math.Max(0, float64(editor.framebufferTransparency)))
	return uint8(transparency * 255)
}

func (editor *Editor) Shutdown() {
	if editor.server != nil {
		editor.server.Close()
	}
	editor.nvim.Close()
	editor.window.Close()
	editor.renderer.Close()
	glfw.Terminate()
}
