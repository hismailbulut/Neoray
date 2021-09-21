package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"

	"github.com/go-gl/glfw/v3.3/glfw"
)

var (
	//go:embed assets/icons/neovim-16.png
	NeovimIconData16x16 []byte
	//go:embed assets/icons/neovim-32.png
	NeovimIconData32x32 []byte
	//go:embed assets/icons/neovim-48.png
	NeovimIconData48x48 []byte
)

// These values are used when setting window state
const (
	WINDOW_SET_STATE_MINIMIZED  = "minimized"
	WINDOW_SET_STATE_MAXIMIZED  = "maximized"
	WINDOW_SET_STATE_FULLSCREEN = "fullscreen"
	WINDOW_SET_STATE_CENTERED   = "centered"
)

type WindowState uint8

// These values are used when specifying current state of the window
const (
	WINDOW_STATE_NORMAL WindowState = iota
	WINDOW_STATE_MINIMIZED
	WINDOW_STATE_MAXIMIZED
	WINDOW_STATE_FULLSCREEN
)

type Window struct {
	handle   *glfw.Window
	title    string
	width    int
	height   int
	dpi      float64
	hasfocus bool

	windowedRect IntRect
	windowState  WindowState
	cursorHidden bool
}

func CreateWindow(width int, height int, title string) Window {
	defer measure_execution_time()()

	assert(width > 0 && height > 0, "Window width or height is smaller than zero.")

	monitor := glfw.GetPrimaryMonitor()
	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Monitor count:", len(glfw.GetMonitors()), "Selected monitor:", monitor.GetName())
	logMessageFmt(LEVEL_DEBUG, TYPE_NEORAY, "Video mode %+v", monitor.GetVideoMode())

	window := Window{title: title}

	// Set opengl library version
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	// We need to create forward compatible context for macos support.
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	if isDebugBuild() {
		glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	}

	// We are initializing window as hidden, and then we show it when mainloop begins
	glfw.WindowHint(glfw.Visible, glfw.False)
	// Framebuffer transparency not working on fullscreen when doublebuffer is on.
	glfw.WindowHint(glfw.DoubleBuffer, glfw.False)
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
	// Scales window width and height to monitor
	glfw.WindowHint(glfw.ScaleToMonitor, glfw.True)

	var err error
	window.handle, err = glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		logMessage(LEVEL_FATAL, TYPE_NEORAY, "Failed to create glfw window:", err)
	}

	// Sometimes windows size is not same with requested size.
	window.width, window.height = window.handle.GetSize()
	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Glfw window created with size", window.width, window.height)

	// Minimum window size
	window.handle.SetSizeLimits(300, 200, glfw.DontCare, glfw.DontCare)

	window.handle.MakeContextCurrent()
	// Disable v-sync, already disabled by default but make sure.
	glfw.SwapInterval(0)

	scaleX, scaleY := window.handle.GetContentScale()
	window.calculateDPI(scaleX, scaleY)

	window.loadDefaultIcons()

	window.handle.SetFramebufferSizeCallback(
		func(w *glfw.Window, width, height int) {
			singleton.window.width = width
			singleton.window.height = height
			// This happens when window minimized.
			if width > 0 && height > 0 {
				rows := height / singleton.cellHeight
				cols := width / singleton.cellWidth
				// Only resize if rows or cols has changed.
				if rows != singleton.renderer.rows || cols != singleton.renderer.cols {
					singleton.nvim.requestResize(rows, cols)
				}
				rglCreateViewport(width, height)
				singleton.render()
			} else {
				// We can pause neoray
			}
		})

	window.handle.SetFocusCallback(
		func(w *glfw.Window, focused bool) {
			window.hasfocus = focused
		})

	window.handle.SetIconifyCallback(
		func(w *glfw.Window, iconified bool) {
			if iconified {
				singleton.window.windowState = WINDOW_STATE_MINIMIZED
			} else {
				singleton.window.windowState = WINDOW_STATE_NORMAL
			}
		})

	window.handle.SetMaximizeCallback(
		func(w *glfw.Window, maximized bool) {
			if maximized {
				singleton.window.windowState = WINDOW_STATE_MAXIMIZED
			} else {
				singleton.window.windowState = WINDOW_STATE_NORMAL
			}
		})

	window.handle.SetRefreshCallback(
		func(w *glfw.Window) {
			defer measure_execution_time()("RefreshCallback")
			// When user resizing the window, glfw.PollEvents call is blocked.
			// And no resizing happens until user releases mouse button. But
			// glfw calls refresh callback and we are additionally updating
			// renderer for resizing the grid or grids. This process may be
			// slow because entire screen redraws in every moment when cell
			// size changed.
			// The update may not render the window, we make sure it will be
			// rendered
			singleton.render()
			singleton.update()
		})

	window.handle.SetContentScaleCallback(
		func(w *glfw.Window, x, y float32) {
			// This function will be called when user changes its content scale
			// in runtime, or moves window to another monitor.
			// First recalculates dpi
			// Second reloads all fonts with same size but different dpi
			// Glfw itself also resizes the window
			singleton.window.calculateDPI(x, y)
			singleton.renderer.setFontSize(0)
		})

	return window
}

func (window *Window) update() {
	if isDebugBuild() {
		fps_string := fmt.Sprintf(" | TPS: %d", singleton.time.lastUPS)
		window.handle.SetTitle(window.title + fps_string)
	}
}

// Argument x and y is window content scale
func (window *Window) calculateDPI(x, y float32) {
	logMessageFmt(LEVEL_DEBUG, TYPE_NEORAY, "Window content scale is: %.2f, %.2f", x, y)
	window.dpi = 96 * float64(y) // We only need Y (?)
}

func (window *Window) loadDefaultIcons() {
	icons := []image.Image{}

	icon48, err := png.Decode(bytes.NewReader(NeovimIconData48x48))
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to decode 48x48 icon:", err)
	} else {
		icons = append(icons, icon48)
	}

	icon32, err := png.Decode(bytes.NewReader(NeovimIconData32x32))
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to decode 32x32 icon:", err)
	} else {
		icons = append(icons, icon32)
	}

	icon16, err := png.Decode(bytes.NewReader(NeovimIconData16x16))
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to decode 16x16 icon:", err)
	} else {
		icons = append(icons, icon16)
	}

	window.handle.SetIcon(icons)
}

func (window *Window) hideCursor() {
	if !window.cursorHidden {
		window.handle.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
		window.cursorHidden = true
	}
}

func (window *Window) showCursor() {
	if window.cursorHidden {
		window.handle.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		window.cursorHidden = false
	}
}

func (window *Window) raise() {
	if window.windowState == WINDOW_STATE_MINIMIZED {
		window.handle.Restore()
		logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Window restored from minimized state.")
	}
	if !window.hasfocus {
		window.handle.Focus()
	}
	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Window raised.")
}

func (window *Window) setState(state string) {
	switch state {
	case WINDOW_SET_STATE_MINIMIZED:
		window.handle.Iconify()
		logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Window state minimized.")
	case WINDOW_SET_STATE_MAXIMIZED:
		window.handle.Maximize()
		logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Window state maximized.")
	case WINDOW_SET_STATE_FULLSCREEN:
		if window.windowState != WINDOW_STATE_FULLSCREEN {
			window.toggleFullscreen()
			// The window losts its input focus after this. We must regain it.
			window.handle.Focus()
		}
	case WINDOW_SET_STATE_CENTERED:
		window.center()
	default:
		logMessage(LEVEL_WARN, TYPE_NEORAY, "Unknown window state:", state)
	}
}

func (window *Window) center() {
	// This may fail on multiple monitors
	videoMode := glfw.GetPrimaryMonitor().GetVideoMode()
	w, h := window.handle.GetSize()
	x := (videoMode.Width / 2) - (w / 2)
	y := (videoMode.Height / 2) - (h / 2)
	window.handle.SetPos(x, y)
	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Window position centered.")
}

func (window *Window) setTitle(title string) {
	window.handle.SetTitle(title)
	window.title = title
}

func (window *Window) setSize(width, height int, inCellSize bool) {
	if inCellSize {
		width *= singleton.cellWidth
		height *= singleton.cellHeight
	}
	if width <= 0 {
		width = window.width
	}
	if height <= 0 {
		height = window.height
	}
	window.handle.SetSize(width, height)
	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Requested window size:", width, height)
}

func (window *Window) toggleFullscreen() {
	if window.handle.GetMonitor() == nil {
		// to fullscreen
		X, Y := window.handle.GetPos()
		W, H := window.handle.GetSize()
		// Store dimension for restoring
		window.windowedRect = IntRect{X: X, Y: Y, W: W, H: H}
		// Fulscreen to current monitor
		monitor := window.getCurrentMonitor(X, Y, W, H)
		videoMode := monitor.GetVideoMode()
		window.handle.SetMonitor(monitor, 0, 0, videoMode.Width, videoMode.Height, videoMode.RefreshRate)
		window.windowState = WINDOW_STATE_FULLSCREEN
	} else {
		// restore
		window.handle.SetMonitor(nil,
			window.windowedRect.X, window.windowedRect.Y,
			window.windowedRect.W, window.windowedRect.H, 0)
		window.windowState = WINDOW_STATE_NORMAL
	}
}

// Returns the monitor where the window currently is.
func (window *Window) getCurrentMonitor(wx, wy, ww, wh int) *glfw.Monitor {
	// Reference:
	// https://stackoverflow.com/a/31526753
	var bestMonitor *glfw.Monitor
	var bestOverlap int
	for _, monitor := range glfw.GetMonitors() {
		mx, my := monitor.GetPos()
		videoMode := monitor.GetVideoMode()
		mw, mh := videoMode.Width, videoMode.Height
		overlap := max(0, min(wx+ww, mx+mw)-max(wx, mx)) * max(0, min(wy+wh, my+mh)-max(wy, my))
		if overlap > bestOverlap {
			bestOverlap = overlap
			bestMonitor = monitor
		}
	}
	assert(bestMonitor != nil, "couldn't find window's monitor")
	return bestMonitor
}

func (window *Window) Close() {
	window.handle.Destroy()
	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Window destroyed.")
}
