package main

import (
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type Options struct {
	// custom options
	cursorAnimTime      float32
	transparency        float32
	targetTPS           int
	contextMenuEnabled  bool
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
	// Grid is a neovim window if multigrid option is enabled, otherwise full
	// screen. Grid has cells and it's attributes, and contains all
	// information how cells and fonts will be rendered.
	// gridManager.go
	gridManager GridManager
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
	// ContextMenu is the only context menu in this program for right click menu.
	// contextmenu.go
	contextMenu ContextMenu
	// Neoray options.
	options Options
	// Tcp server for singleinstance
	// tcp.go
	server *TCPServer
	// If quitRequested is true the program will quit.
	quitRequested chan bool
	// Initializing in CreateRenderer
	// TODO: I am going to implement per grid font size, and these variables will be moved to grid.
	cellWidth  int
	cellHeight int
	// Mainloop timing values
	time struct {
		ticker   *time.Ticker
		interval time.Duration
		lastTick time.Time
		delta    float64
		lastUPS  int
	}
}

func (editor *Editor) Initialize() {
	editor.quitRequested = make(chan bool)

	editor.nvim = CreateNvimProcess()
	editor.nvim.startUI()

	editor.initGlfw()
	editor.window = CreateWindow(WINDOW_SIZE_AUTO, WINDOW_SIZE_AUTO, TITLE)

	initInputEvents()

	editor.uiOptions = UIOptions{}

	editor.gridManager = CreateGridManager()
	editor.mode = CreateMode()
	editor.cursor = CreateCursor()
	editor.contextMenu = CreateContextMenu()
	editor.renderer = CreateRenderer()

	editor.options = CreateDefaultOptions()
	editor.nvim.requestStartupVariables()

	// show the main window
	editor.window.handle.Show()
	logDebug("Window is now visible.")
}

func CreateDefaultOptions() Options {
	return Options{
		cursorAnimTime:      0.06,
		transparency:        1,
		targetTPS:           60,
		contextMenuEnabled:  true,
		keyToggleFullscreen: "<F11>",
		keyIncreaseFontSize: "<C-kPlus>",
		keyDecreaseFontSize: "<C-kMinus>",
		mouseHide:           false,
	}
}

func (editor *Editor) initGlfw() {
	defer measure_execution_time()()
	if err := glfw.Init(); err != nil {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize glfw:", err)
	}
	logMessage(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Glfw version:", glfw.GetVersionString())
}

func (editor *Editor) MainLoop() {
	// For measuring total time of the program.
	programBegin := time.Now()
	// Ticker's interval
	editor.time.interval = time.Second / time.Duration(editor.options.targetTPS)
	// NOTE: Ticker is not working correctly on windows.
	editor.time.ticker = time.NewTicker(editor.time.interval)
	defer editor.time.ticker.Stop()
	// For measuring delta time
	upsTimer := 0.0
	updates := 0
	// For measuring elpased time
	editor.time.lastTick = time.Now()
	// Mainloop
	run := true
	for run {
		select {
		case tick := <-editor.time.ticker.C:
			// Calculate delta time
			elapsed := tick.Sub(editor.time.lastTick)
			editor.time.lastTick = tick
			editor.time.delta = elapsed.Seconds()
			// Increment counters
			upsTimer += editor.time.delta
			updates++
			// Calculate updates per second
			if upsTimer >= 1 {
				editor.time.lastUPS = updates
				updates = 0
				upsTimer -= 1
			}
			// Update program
			editor.update()
			glfw.PollEvents()
			// Check for window close
			if editor.window.handle.ShouldClose() {
				// Send quit command to neovim and not quit until neovim quits.
				editor.window.handle.SetShouldClose(false)
				go editor.nvim.executeVimScript("qa")
			}
		case <-editor.quitRequested:
			run = false
		}
	}
	logMessage(LOG_LEVEL_TRACE, LOG_TYPE_PERFORMANCE, "Program finished. Total execution time:", time.Since(programBegin))
}

func (editor *Editor) update() {
	// Order is important!
	handleRedrawEvents()
	editor.window.update()
	editor.cursor.update()
	editor.renderer.update()
	editor.nvim.update()
	if editor.server != nil {
		editor.server.update()
	}
}

func (editor *Editor) resetTicker() {
	editor.time.interval = time.Second / time.Duration(editor.options.targetTPS)
	editor.time.ticker.Reset(editor.time.interval)
}

func (editor *Editor) backgroundAlpha() uint8 {
	return uint8(editor.options.transparency * 255)
}

// If this function called, the screen will be rendered in current loop.
func (editor *Editor) render() {
	editor.renderer.renderCall = true
}

// If this function called, the entire screen will be drawed in current loop.
func (editor *Editor) fullDraw() {
	editor.renderer.fullDrawCall = true
}

// If this function called, changed cells will be drawed in current loop.
func (editor *Editor) draw() {
	editor.renderer.drawCall = true
}

// This function prints cell at the pos.
func (editor *Editor) debugPrintCell(pos IntVec2) {
	id, x, y := editor.gridManager.getCellAt(pos)
	if !editorParsedArgs.multiGrid {
		id = 1
	}
	grid := editor.gridManager.grids[id]
	cell := grid.getCell(x, y)
	vertex := editor.renderer.debugGetCellData(grid.sRow+x, grid.sCol+y)
	format := `Cell information:
	grid: %s
	pos: %d %d
	char: %s %d
	attrib_id: %d
	needs_redraw: %t
	data : %+v`
	logfDebug(format, grid, x, y, string(cell.char), cell.char, cell.attribId, cell.needsDraw, vertex)
}

func (editor *Editor) Shutdown() {
	if editor.server != nil {
		editor.server.Close()
	}
	editor.nvim.Close()
	editor.renderer.Close()
	editor.window.Close()
	glfw.Terminate()
	logDebug("Glfw terminated.")
}
