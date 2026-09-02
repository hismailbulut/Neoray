package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/fontkit"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/hismailbulut/Neoray/pkg/window"

	"github.com/hismailbulut/Neoray/cmd/neoray/assets"
)

type Options struct {
	// custom options
	cursorAnimTime      float32
	transparency        float32
	targetTPS           int
	contextMenuEnabled  bool
	boxDrawingEnabled   bool
	imageViewerEnabled  bool
	keyToggleFullscreen string
	keyIncreaseFontSize string
	keyDecreaseFontSize string
}

func DefaultOptions() Options {
	return Options{
		cursorAnimTime:      0.1,
		transparency:        1,
		targetTPS:           60,
		contextMenuEnabled:  true,
		boxDrawingEnabled:   true,
		imageViewerEnabled:  true,
		keyToggleFullscreen: "<F11>",
		keyIncreaseFontSize: "<C-kPlus>",
		keyDecreaseFontSize: "<C-kMinus>",
	}
}

type EditorState uint32

const (
	EditorNotInitialized EditorState = iota
	EditorInitialized
	EditorLoopStarted // Mainloop started and app running, but not everything ready because of neovim events processed at mainloop
	EditorFirstFlush  // This is where we start to check NeoraySet options
	EditorWindowShown // We show window after first NeoraySet option check
	EditorLoopStopped
	EditorDestroyed
)

func (state EditorState) String() string {
	switch state {
	case EditorNotInitialized:
		return "EditorNotInitialized"
	case EditorInitialized:
		return "EditorInitialized"
	case EditorLoopStarted:
		return "EditorLoopStarted"
	case EditorFirstFlush:
		return "EditorFirstFlush"
	case EditorWindowShown:
		return "EditorWindowShown"
	case EditorLoopStopped:
		return "EditorLoopStopped"
	case EditorDestroyed:
		return "EditorDestroyed"
	}
	panic("unknown editor state")
}

type Ticker struct {
	ticker     *time.Ticker
	ticks      int
	draws      int
	forceDraws int
	renders    int
}

func (tickCounter *Ticker) ResetTicker(targetTPS int) {
	if tickCounter.ticker == nil {
		tickCounter.ticker = time.NewTicker(time.Second / time.Duration(targetTPS))
	} else {
		tickCounter.ticker.Reset(time.Second / time.Duration(targetTPS))
	}
}

func (tickCounter *Ticker) ResetCounts() {
	tickCounter.ticks = 0
	tickCounter.draws = 0
	tickCounter.forceDraws = 0
	tickCounter.renders = 0
}

type Editor struct {
	state EditorState
	// Parsed startup arguments
	parsedArgs ParsedArgs
	// IPC server for singleinstance
	server *IpcServer
	// Neoray options.
	options Options
	// Main window of this program.
	window *window.Window
	// Input manager handles all keyboard and mouse inputs
	inputManager *InputManager
	// FontKitStack holds our fonts
	fontManager *FontManager
	// Grid manager holds information about neovim grids and how they will be rendered
	// We also use its underlying rendering structure when rendering cursor and context menu
	gridManager *GridManager
	// Cursor represents a neovim cursor and all it's information
	cursor *Cursor
	// ContextMenu is the only context menu in this program for right click menu.
	contextMenu *ContextMenu
	// ImageViewer
	imageViewer *ImageViewer
	// UIOptions is a struct, holds some user ui uiOptions like guifont.
	uiOptions UIOptions
	// Neovim child process
	nvim *NvimProcess
	// MainLoop ticker
	tickCounter Ticker
	// Stops mainloop
	quitChan chan bool
	// Draw calls
	cDraw      bool
	cForceDraw bool
	cRender    bool
}

func NewEditor() *Editor {
	editor := &Editor{
		quitChan: make(chan bool, 1),
	}
	return editor
}

func (editor *Editor) Init() error {
	editor.options = DefaultOptions()
	err := glfw.Init()
	if err != nil {
		return fmt.Errorf("Failed to initialize GLFW3: %s", err)
	}
	logger.Trace("GLFW3 Version:", glfw.GetVersionString())
	editor.window, err = window.New(NAME, 800, 600, bench.IsDebugBuild())
	if err != nil {
		return fmt.Errorf("Failed to create window: %s", err)
	}
	// Event handler function runs when we call window.PollEvents
	editor.window.SetEventHandler(editor.EventHandler)
	// Set window minimum size
	editor.window.SetMinSize(common.Vec2(300, 200))
	// Set window icons
	editor.LoadDefaultIcons()
	// Update opengl viewport
	editor.window.GL().SetViewport(editor.window.Viewport())
	// Print some opengl info
	info := editor.window.GL().Info()
	logger.Trace("Opengl Version:", info.Version)
	logger.Trace("Vendor:", info.Vendor)
	logger.Trace("Renderer:", info.Renderer)
	logger.Trace("GLSL:", info.ShadingLanguageVersion)
	logger.Trace("Max Texture Size:", info.MaxTextureSize)
	// Initialize input manager
	editor.inputManager = NewInputManager(editor)
	// Set default font
	editor.fontManager = NewFontManager(editor)
	// Initialize gridManager
	editor.gridManager = NewGridManager(editor)
	// Initialize cursor
	editor.cursor = NewCursor(editor)
	// Initialize contextMenu
	editor.contextMenu = NewContextMenu(editor)
	// Initialize imageViewer
	editor.imageViewer = NewImageViewer(editor)
	// TODO Move this to gridManager
	editor.uiOptions = CreateUIOptions(editor)
	// Start neovim
	editor.nvim = CreateNvimProcess(editor)
	// Calculate temporary start size and start the ui connection
	// The size will be updated according to user preferences
	cellSize := editor.DefaultCellSize()
	cols := editor.window.Size().Width() / cellSize.Width()
	rows := editor.window.Size().Height() / cellSize.Height()
	logger.Debug("Calculated startup size of the neovim is", rows, cols)
	editor.nvim.StartUI(rows, cols)
	// Initialization done
	editor.SetState(EditorInitialized)
	return nil
}

func (editor *Editor) ProcessArgsBeforeInit() bool {
	parsedArgs, err, quit := ParseArgs(os.Args[1:])
	if err != nil {
		logger.Fatal(err)
	}
	if quit {
		return true
	}
	editor.parsedArgs = parsedArgs
	return editor.parsedArgs.ProcessBefore()
}

func (editor *Editor) ProcessArgsAfterInit() {
	editor.parsedArgs.ProcessAfter(editor)
}

func (editor *Editor) LoadDefaultIcons() {
	icons := [3]image.Image{}
	icon48, err := png.Decode(bytes.NewReader(assets.NeovimIconData48x48))
	if err != nil {
		logger.Error("Failed to decode 48x48 icon:", err)
	} else {
		icons[0] = icon48
	}

	icon32, err := png.Decode(bytes.NewReader(assets.NeovimIconData32x32))
	if err != nil {
		logger.Error("Failed to decode 32x32 icon:", err)
	} else {
		icons[1] = icon32
	}

	icon16, err := png.Decode(bytes.NewReader(assets.NeovimIconData16x16))
	if err != nil {
		logger.Error("Failed to decode 16x16 icon:", err)
	} else {
		icons[2] = icon16
	}
	editor.window.SetIcon(icons)
}

// A helper function, if default grid is not set by neovim yet we use this for cell size
func (editor *Editor) DefaultCellSize() common.Vector2[int] {
	face, _ := editor.fontManager.DefaultFont(false, false).CreateFace(fontkit.FaceParams{
		Size:            DEFAULT_FONT_SIZE,
		DPI:             editor.window.DPI(),
		UseBoxDrawing:   false,
		UseBlockDrawing: false,
	})
	return face.ImageSize()
}

func (editor *Editor) ResizeWindowInCellFormat(rows, cols int) {
	var size common.Vector2[int]
	defaultGrid := editor.gridManager.Grid(1)
	if defaultGrid != nil {
		size.X = cols * defaultGrid.CellSize().Width()
		size.Y = rows * defaultGrid.CellSize().Height()
	} else {
		cellSize := editor.DefaultCellSize()
		size.X = cols * cellSize.Width()
		size.Y = rows * cellSize.Height()
	}
	editor.window.Resize(size)
}

// This is for making sure the state changing valid
func (editor *Editor) SetState(state EditorState) {
	if editor.state >= state {
		logger.Fatalf("Editor state can only be incremented")
	}
	editor.state = state
	logger.Debug("Editor state changed to", state)
}

// Shows the window if it is not visible yet
// Does nothing if window is already shown
func (editor *Editor) ShowWindow() {
	if editor.state+1 == EditorWindowShown {
		editor.window.Show()
		editor.SetState(EditorWindowShown)
		logger.Trace("Window is visible now in", time.Since(StartTime))
		// Currently neovim sends a default guifont option even the user has its own guifont
		// because of this we do not load fonts immediately and wait for changes until window
		// has to be shown
		editor.fontManager.LoadFonts()
	}
}

func (editor *Editor) MarkDraw() {
	editor.cDraw = true
}

func (editor *Editor) MarkForceDraw() {
	editor.cForceDraw = true
}

func (editor *Editor) MarkRender() {
	editor.cRender = true
}

func (editor *Editor) SetTargetTPS(targetTPS int) {
	editor.options.targetTPS = targetTPS
	editor.tickCounter.ResetTicker(targetTPS)
}

func (editor *Editor) MainLoop() {
	editor.SetState(EditorLoopStarted)
	editor.SetTargetTPS(editor.options.targetTPS)
	// For measuring total time of the program.
	programBegin := time.Now()
	// For measuring ticks per second, debugging purposes
	upsTimer := 0.0
	// For measuring elpased time
	lastTick := time.Now()
	// Mainloop
	run := true
	for run {
		select {
		case tick := <-editor.tickCounter.ticker.C:
			// Calculate delta time
			elapsed := tick.Sub(lastTick)
			lastTick = tick
			delta := elapsed.Seconds()
			// Increment counters
			upsTimer += delta
			// Calculate updates per second
			if upsTimer >= 1 {
				/*logger.Debug(
				"TPS:", editor.tickCounter.ticks,
				"DPS:", editor.tickCounter.draws,
				"FDPS:", editor.tickCounter.forceDraws,
				"RPS:", editor.tickCounter.renders)*/
				editor.tickCounter.ResetCounts()
				upsTimer -= 1
			}
			// Handle with inputs first
			editor.window.PollEvents()
			// then update
			editor.UpdateHandler(float32(delta))
		case <-editor.quitChan:
			run = false
		}
	}
	editor.SetState(EditorLoopStopped)
	logger.Trace("Program finished. Total execution time:", time.Since(programBegin))
}

func (editor *Editor) UpdateHandler(delta float32) {
	editor.tickCounter.ticks++
	// Update required stuff
	editor.nvim.Update()
	editor.gridManager.Update()
	editor.cursor.Update(delta)
	editor.imageViewer.Update()
	if editor.server != nil {
		editor.server.Update()
	}
	// Draw and render
	if editor.state >= EditorWindowShown {
		// We have to collect draw calls and zero them before drawing things because
		// the draw functions may also set a draw call for next tick
		draw := false
		forceDraw := false
		render := false
		if editor.cDraw {
			draw = true
			editor.tickCounter.draws++
			editor.cDraw = false
		}
		if editor.cForceDraw {
			forceDraw = true
			editor.tickCounter.forceDraws++
			editor.cForceDraw = false
		}
		if editor.cRender {
			render = true
			editor.tickCounter.renders++
			editor.cRender = false
		}
		// Draw calls
		if draw || forceDraw {
			EndBenchmark := bench.Begin()
			editor.gridManager.Draw(forceDraw)
			editor.cursor.Draw(delta)
			editor.contextMenu.Draw()
			editor.imageViewer.Draw()
			EndBenchmark("UpdateHandler.Draw")
			render = true
		}
		// Render calls
		if render {
			EndBenchmark := bench.Begin()
			// Clear background
			bg := editor.gridManager.background
			bg.A = editor.options.transparency
			editor.window.GL().ClearScreen(bg)
			// Render in order
			editor.gridManager.Render()
			editor.cursor.Render()
			editor.contextMenu.Render()
			editor.imageViewer.Render()
			// Flush to make changes visible
			editor.window.GL().Flush()
			EndBenchmark("UpdateHandler.Render")
		}
	}
}

func (editor *Editor) EventHandler(event window.WindowEvent) {
	switch event.Type {
	case window.WindowEventRefresh:
		{
			// Eg. When user resizing the window, glfw.PollEvents call is blocked.
			// And no events receives except this one. We need to update Neoray
			// additionally when refresh event received.
			// Only send if it not received already
			size := editor.window.Size()
			editor.EventHandler(window.WindowEvent{
				Type:   window.WindowEventResize,
				Params: []any{size.Width(), size.Height()},
			})
			// Only update if tick received
			select {
			case <-editor.tickCounter.ticker.C:
				// TODO: calculate delta
				editor.UpdateHandler(0)
			default:
			}
		}
	case window.WindowEventResize:
		{
			// Check grids sizes
			width := event.Params[0].(int)
			height := event.Params[1].(int)
			// When window minimized, glfw sends a resize event with zero size
			if width <= 0 || height <= 0 {
				break
			}
			// Update viewport
			editor.window.GL().SetViewport(editor.window.Viewport())
			// Mark render because viewport changed
			editor.MarkRender()
			// Update grid size
			editor.gridManager.CheckDefaultGridSize()
			/*
				defaultGrid := editor.gridManager.Grid(1)
				if defaultGrid == nil {
					break
				}
				cellSize := defaultGrid.CellSize()
				rows := height / cellSize.Height()
				cols := width / cellSize.Width()
				if rows == defaultGrid.rows && cols == defaultGrid.cols {
					break
				}
				// Try to resize the neovim
				editor.nvim.TryResizeUI(rows, cols)
			*/
		}
	case window.WindowEventKeyInput:
		{
			key := event.Params[0].(glfw.Key)
			scancode := event.Params[1].(int)
			action := event.Params[2].(glfw.Action)
			mods := event.Params[3].(glfw.ModifierKey)
			editor.inputManager.KeyInputHandler(key, scancode, action, mods)
		}
	case window.WindowEventCharInput:
		{
			char := event.Params[0].(rune)
			editor.inputManager.CharInputHandler(char)
		}
	case window.WindowEventMouseInput:
		{
			button := event.Params[0].(glfw.MouseButton)
			action := event.Params[1].(glfw.Action)
			mods := event.Params[2].(glfw.ModifierKey)
			editor.inputManager.MouseInputHandler(button, action, mods)
		}
	case window.WindowEventMouseMove:
		{
			xpos := event.Params[0].(float64)
			ypos := event.Params[1].(float64)
			editor.inputManager.MouseMoveHandler(xpos, ypos)
		}
	case window.WindowEventScroll:
		{
			xoff := event.Params[0].(float64)
			yoff := event.Params[1].(float64)
			editor.inputManager.ScrollHandler(xoff, yoff)
		}
	case window.WindowEventDrop:
		{
			files := event.Params[0].([]string)
			editor.inputManager.DropHandler(files)
		}
	case window.WindowEventScaleChanged:
		{
			editor.gridManager.ResetFontSize()
		}
	case window.WindowEventClose:
		{
			if editor.nvim.connectedViaTcp {
				// Neoray is not responsible for closing neovim.
				editor.nvim.Disconnect()
				// Stop loop
				editor.quitChan <- true
			} else {
				// Send quit command to neovim and wait until neovim quits.
				editor.window.KeepAlive()
				go editor.nvim.Command("qa")
			}
		}
	}
}

func (editor *Editor) Terminate() {
	editor.tickCounter.ticker.Stop()
	if editor.server != nil {
		editor.server.Close()
	}
	editor.nvim.Close()
	editor.imageViewer.Destroy()
	editor.contextMenu.Destroy()
	editor.cursor.Destroy()
	editor.fontManager.Destroy()
	editor.gridManager.Destroy()
	editor.window.Destroy()
	glfw.Terminate()
	editor.SetState(EditorDestroyed) // This is actually unnecessary
	logger.Debug("Editor terminated")
}
