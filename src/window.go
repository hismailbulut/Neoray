package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	WINDOW_STATE_MINIMIZED  = "minimized"
	WINDOW_STATE_MAXIMIZED  = "maximized"
	WINDOW_STATE_FULLSCREEN = "fullscreen"
	WINDOW_STATE_CENTERED   = "centered"

	WINDOW_SIZE_AUTO = 1 << 30
)

type Window struct {
	handle *glfw.Window
	title  string
	width  int
	height int
	dpi    float64

	// internal usage
	windowedRect IntRect
	minimized    bool
	fullscreen   bool
	cursorHidden bool
}

func CreateWindow(width int, height int, title string) Window {
	defer measure_execution_time()()

	logDebug("Creating glfw window.")

	videoMode := glfw.GetPrimaryMonitor().GetVideoMode()
	rW := videoMode.Width
	rH := videoMode.Height

	if width == WINDOW_SIZE_AUTO {
		width = (rW / 5) * 3
	}
	if height == WINDOW_SIZE_AUTO {
		height = (rH / 4) * 3
	}

	logDebug("Window width:", width, "height:", height)

	window := Window{
		title:  title,
		width:  width,
		height: height,
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)

	// Framebuffer transparency not working on fullscreen when doublebuffer is on.
	glfw.WindowHint(glfw.DoubleBuffer, glfw.False)

	windowHandle, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to create glfw window:", err)
	}

	logDebug("Glfw window created successfully.")
	window.handle = windowHandle

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
			}
		})

	window.handle.SetIconifyCallback(
		func(w *glfw.Window, iconified bool) {
			singleton.window.minimized = iconified
		})

	window.handle.SetRefreshCallback(
		func(w *glfw.Window) {
			defer measure_execution_time()("RefreshCallback")
			// When user resizing the window, glfw.PollEvents call is blocked.
			// And no resizing happens until user releases mouse button. But
			// glfw calls refresh callback and we are additionally updating
			// renderer for resizing the grid or grids. This process is very
			// slow because entire screen redraws in every moment when cell
			// size changed.
			singleton.update()
		})

	window.calculateDPI()

	window.handle.MakeContextCurrent()

	return window
}

func (window *Window) update() {
	if isDebugBuild() {
		fps_string := fmt.Sprintf(" | TPS: %d | Delta: %f",
			singleton.updatesPerSecond, singleton.deltaTime)
		idx := strings.LastIndex(window.title, " | TPS:")
		if idx == -1 {
			window.setTitle(window.title + fps_string)
		} else {
			window.setTitle(window.title[0:idx] + fps_string)
		}
	}
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
	if window.minimized {
		window.handle.Restore()
	}
	// TODO
	window.handle.SetAttrib(glfw.Floating, glfw.True)
	window.handle.SetAttrib(glfw.Floating, glfw.False)
}

func (window *Window) setState(state string) {
	switch state {
	case WINDOW_STATE_MINIMIZED:
		window.handle.Iconify()
		break
	case WINDOW_STATE_MAXIMIZED:
		window.handle.Maximize()
		break
	case WINDOW_STATE_FULLSCREEN:
		if !window.fullscreen {
			window.toggleFullscreen()
		}
		break
	case WINDOW_STATE_CENTERED:
		window.center()
		break
	default:
		logMessage(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Unknown window state:", state)
		break
	}
}

func (window *Window) center() {
	mode := glfw.GetPrimaryMonitor().GetVideoMode()
	w, h := window.handle.GetSize()
	x := (mode.Width / 2) - (w / 2)
	y := (mode.Height / 2) - (h / 2)
	window.handle.SetPos(x, y)
}

func (window *Window) setSize(width, height int) {
	window.handle.SetSize(width, height)
}

func (window *Window) setTitle(title string) {
	window.handle.SetTitle(title)
	window.title = title
}

func (window *Window) toggleFullscreen() {
	if window.handle.GetMonitor() == nil {
		// to fullscreen
		x, y := window.handle.GetPos()
		w, h := window.handle.GetSize()
		window.windowedRect = IntRect{X: x, Y: y, W: w, H: h}
		monitor := glfw.GetPrimaryMonitor()
		videoMode := monitor.GetVideoMode()
		window.handle.SetMonitor(monitor, 0, 0,
			videoMode.Width, videoMode.Height, videoMode.RefreshRate)
		window.fullscreen = true
	} else {
		// restore
		window.handle.SetMonitor(nil,
			window.windowedRect.X, window.windowedRect.Y,
			window.windowedRect.W, window.windowedRect.H, 0)
		window.fullscreen = false
	}
}

func (window *Window) calculateDPI() {
	monitor := glfw.GetPrimaryMonitor()

	// Calculate physical diagonal size of the monitor in inches
	pWidth, pHeight := monitor.GetPhysicalSize() // returns size in millimeters
	pDiagonal := math.Sqrt(float64(pWidth*pWidth+pHeight*pHeight)) * 0.0393700787

	// Calculate logical diagonal size of the monitor in pixels
	scaleX, scaleY := monitor.GetContentScale()
	mWidth := float64(monitor.GetVideoMode().Width) * float64(scaleX)
	mHeight := float64(monitor.GetVideoMode().Height) * float64(scaleY)
	mDiagonal := math.Sqrt(mWidth*mWidth + mHeight*mHeight)

	// Calculate dpi
	window.dpi = mDiagonal / pDiagonal
	if window.dpi < 72 {
		// This could be actual dpi or we may failed to calculate dpi.
		logMessage(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Device dpi", window.dpi, "is very low and automatically set to 72.")
		window.dpi = 72
	}
	logDebug("Monitor diagonal:", pDiagonal, "dpi:", window.dpi)
}

func (window *Window) Close() {
	window.handle.Destroy()
}
