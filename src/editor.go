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
	// PopupMenu is the only popup menu in this program for right click menu.
	// popupmenu.go
	popupMenu PopupMenu
	// Neoray options.
	options Options
	// Tcp server for singleinstance
	// tcp.go
	server *TCPServer
	// If quitRequested is true the program will quit.
	quitRequested chan bool
	// This is for resizing from nvim side, first we will send resize request
	// to neovim, than we wait for the resize call from neovim side. When
	// waiting this we dont want to render.
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
	// Last elapsed time between updates.
	deltaTime float64
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
	editor.popupMenu = CreatePopupMenu()
	editor.renderer = CreateRenderer()

	editor.options = CreateDefaultOptions()
	editor.nvim.requestOptions()
}

func CreateDefaultOptions() Options {
	return Options{
		cursorAnimTime:      0.06,
		transparency:        1,
		targetTPS:           60,
		popupMenuEnabled:    true,
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
	interval := time.Second / time.Duration(editor.options.targetTPS)
	// NOTE: Ticker is not working correctly on windows.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// For measuring tps
	var secondTimer float64
	updates := 0
	// For measuring elpased time
	prevTick := time.Now()
	// Mainloop
	run := true
	for run {
		select {
		case tick := <-ticker.C:
			// Calculate delta time
			elapsed := tick.Sub(prevTick)
			prevTick = tick
			editor.deltaTime = elapsed.Seconds()
			// Increment counters
			secondTimer += editor.deltaTime
			updates++
			// Calculate updates per second
			if secondTimer >= 1 {
				editor.updatesPerSecond = updates
				updates = 0
				secondTimer -= 1
			}
			// Update program
			editor.update()
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
	if editor.server != nil {
		editor.server.update()
	}
	handleRedrawEvents()
	if !editor.waitingResize {
		editor.window.update()
		editor.cursor.update()
		editor.renderer.update()
	}
	glfw.PollEvents()
}

// Calculates dimensions of the window as cell size.
// Returns true if the dimensions has changed.
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

func (editor *Editor) debugEvalCell(x, y int) {
	// TODO Won't work since multigrid.
	// cell := editor.gridManager.getCell(1, x, y)
	// vertex := editor.renderer.debugGetCellData(x, y)
	// format := `Cell information:
	// pos: %d %d
	// char: %s %d
	// attrib_id: %d
	// needs_redraw: %t
	// Data : %+v`
	// logfDebug(format, x, y, string(cell.char), cell.char, cell.attribId, cell.needsDraw, vertex)
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
