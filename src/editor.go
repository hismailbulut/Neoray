package main

import (
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	MINIMUM_FONT_SIZE = 7
	DEFAULT_FONT_SIZE = 12
)

type Options struct {
	// custom options
	cursorAnimTime      float32
	transparency        float32
	targetTPS           int
	popupMenuEnabled    bool
	keyToggleFullscreen string
	keyIncreaseFontSize string
	keyDecreaseFontSize string
	// builtin options
	mouseHide bool
}

type Editor struct {
	// Neovim child process
	// nvim_process.go
	nvim NvimProcess
	// Main window of this program.
	// window.go
	window Window
	// Grid is a neovim window if multigrid option is enabled, otherwise full screen.
	// Grid has cells and it's attributes, and contains all information how cells and fonts will be rendered.
	// grid.go
	grid Grid
	// Cursor represents a neovim cursor and all it's information
	// cursor.go
	cursor Cursor
	// Mode is current nvim mode information struct (normal, visual etc.)
	// We need for cursor rendering
	// mode.go
	mode Mode
	// Renderer is responsible for holding and oragnizing rendering data and
	// sending them to opengl.
	// renderer.go, opengl calls is in renderergl.go
	renderer Renderer
	// UIOptions is a struct, holds some user ui uiOptions like guifont.
	// uioptions.go
	uiOptions UIOptions
	// PopupMenu is the only popup menu in this program for right click menu.
	// popupmenu.go
	popupMenu PopupMenu
	// Neoray options.
	options Options
	// Tcp server for singleinstance
	// tcp.go
	server *TCPServer

	// These are the global variables of neoray. They are same in everywhere.
	// Some of them are initialized at runtime. Therefore we must be carefull when we
	// use them. If you add more here just write some information about it.
	// If quitRequested is true the program will quit.
	quitRequestedChan chan bool
	// This is for resizing from nvim side, first we will send resize request to neovim,
	// than we wait for the resize call from neovim side. When waiting this we dont want
	// to render.
	waitingResize bool
	// Initializing in CreateRenderer
	cellWidth  int
	cellHeight int
	// Initializing in Editor.calculateCellCount
	rowCount    int
	columnCount int
	cellCount   int
	// Initializing in Editor.MainLoop
	updatesPerSecond int
	// For debugging.
	averageTPS float32
	// Last elapsed time between updates.
	deltaTime float32
}

func (editor *Editor) Initialize() {
	defer measure_execution_time()()

	editor.options = CreateDefaultOptions()
	editor.nvim = CreateNvimProcess()

	if err := glfw.Init(); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize glfw:", err)
	}
	log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Glfw version:", glfw.GetVersionString())

	editor.window = CreateWindow(WINDOW_SIZE_AUTO, WINDOW_SIZE_AUTO, TITLE)
	InitializeInputEvents()

	editor.grid = CreateGrid()
	editor.mode = CreateMode()
	editor.cursor = CreateCursor()
	editor.popupMenu = CreatePopupMenu()
	editor.uiOptions = UIOptions{}

	editor.renderer = CreateRenderer()

	editor.quitRequestedChan = make(chan bool)
	editor.nvim.startUI()
}

func CreateDefaultOptions() Options {
	return Options{
		cursorAnimTime:      0.8,
		transparency:        1,
		targetTPS:           60,
		popupMenuEnabled:    true,
		keyToggleFullscreen: "<F11>",
		keyIncreaseFontSize: "<C-kPlus>",
		keyDecreaseFontSize: "<C-kMinus>",
		mouseHide:           false,
	}
}

func (editor *Editor) MainLoop() {
	// For measuring total time of the program.
	programBegin := time.Now()
	// Ticker's interval
	interval := time.Second / time.Duration(editor.options.targetTPS)
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
				editor.updatesPerSecond = ticks
				editor.averageTPS = (editor.averageTPS + float32(editor.updatesPerSecond)) / 2
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
		// Instead of immediately closing we will send simple
		// quit command to neovim and if there are unsaved files
		// the neovim will handle them and user will not lose its progress.
		editor.window.handle.SetShouldClose(false)
		go editor.nvim.executeVimScript("qa")
		goto MAINLOOP
	}
	log_debug("Average TPS:", editor.averageTPS)
	log_message(LOG_LEVEL_TRACE, LOG_TYPE_PERFORMANCE, "Program finished. Total execution time:", time.Since(programBegin))
}

// Calculates dimensions of the window as cell size.
// Returns true if the cell count or rows or columns changed.
func (editor *Editor) calculateCellCount() bool {
	cols := editor.window.width / editor.cellWidth
	rows := editor.window.height / editor.cellHeight
	if cols != editor.columnCount || rows != editor.rowCount {
		editor.columnCount = cols
		editor.rowCount = rows
		editor.cellCount = cols * rows
		return true
	}
	return false
}

func (editor *Editor) backgroundAlpha() uint8 {
	return uint8(editor.options.transparency * 255)
}

func (editor *Editor) render() {
	editor.renderer.renderCall = true
}

func (editor *Editor) draw() {
	editor.renderer.drawCall = true
}

func (editor *Editor) debugEvalCell(x, y int) {
	cell := editor.grid.getCell(x, y)
	vertices := editor.renderer.debugGetCellData(x, y)
	format := `Cell information:
	pos: %d %d
	char: %s %d
	attrib_id: %d
	needs_redraw: %t
	Data 0: %+v
	Data 1: %+v
	Data 2: %+v
	Data 3: %+v`
	logf_debug(format, x, y, string(cell.char), cell.char,
		cell.attribId, cell.needsDraw,
		vertices[0], vertices[1], vertices[2], vertices[3])
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
