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

	videoMode := glfw.GetPrimaryMonitor().GetVideoMode()
	logfDebug("Video mode %+v", videoMode)

	if width == WINDOW_SIZE_AUTO {
		width = (videoMode.Width / 5) * 3
	}
	if height == WINDOW_SIZE_AUTO {
		height = (videoMode.Height / 4) * 3
	}

	logDebug("Creating window, width:", width, "height:", height)

	window := Window{
		title:  title,
		width:  width,
		height: height,
	}

	// Set opengl library version
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	// We need to create forward compatible context for macos support.
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	if isDebugBuild() {
		glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	}

	// We are initializing window as hidden, when the mainloop is started, window will be shown.
	glfw.WindowHint(glfw.Visible, glfw.False)
	// Framebuffer transparency not working on fullscreen when doublebuffer is on.
	glfw.WindowHint(glfw.DoubleBuffer, glfw.False)
	// NOTE: On some systems, framebuffer transparency causes unexpected shutdown or blank screen.
	// We need to be able to enable/disable this in release build.
	// Or maybe there is something I did wrong.
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)

	var err error
	window.handle, err = glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to create glfw window:", err)
	}
	logDebug("Glfw window created successfully.")

	window.handle.MakeContextCurrent()

	// Disable v-sync
	glfw.SwapInterval(0)

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

	window.handle.SetContentScaleCallback(
		func(w *glfw.Window, x, y float32) {
			singleton.window.calculateDPI()
			singleton.renderer.setFontSize(0)
		})

	window.calculateDPI()

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
		logDebug("Window restored from minimized state.")
	}
	// TODO
	window.handle.SetAttrib(glfw.Floating, glfw.True)
	window.handle.SetAttrib(glfw.Floating, glfw.False)
	logDebug("Window raised.")
}

func (window *Window) setState(state string) {
	switch state {
	case WINDOW_STATE_MINIMIZED:
		window.handle.Iconify()
		logDebug("Window state minimized.")
	case WINDOW_STATE_MAXIMIZED:
		window.handle.Maximize()
		logDebug("Window state maximized.")
	case WINDOW_STATE_FULLSCREEN:
		if !window.fullscreen {
			window.toggleFullscreen()
		}
	case WINDOW_STATE_CENTERED:
		window.center()
	default:
		logMessage(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Unknown window state:", state)
	}
}

func (window *Window) center() {
	videoMode := glfw.GetPrimaryMonitor().GetVideoMode()
	w, h := window.handle.GetSize()
	x := (videoMode.Width / 2) - (w / 2)
	y := (videoMode.Height / 2) - (h / 2)
	window.handle.SetPos(x, y)
	logDebug("Window position centered.")
}

func (window *Window) setTitle(title string) {
	window.handle.SetTitle(title)
	window.title = title
}

func (window *Window) toggleFullscreen() {
	if window.handle.GetMonitor() == nil {
		// to fullscreen
		X, Y := window.handle.GetPos()
		W, H := window.handle.GetSize()
		window.windowedRect = IntRect{X: X, Y: Y, W: W, H: H}
		monitor := glfw.GetPrimaryMonitor()
		videoMode := monitor.GetVideoMode()
		window.handle.SetMonitor(monitor, 0, 0, videoMode.Width, videoMode.Height, videoMode.RefreshRate)
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

	// Get content scale, there are two of them and I don't know which one is
	// to use, and I decided to use average of them
	msx, msy := monitor.GetContentScale()
	wsx, wsy := window.handle.GetContentScale()
	scaleX := (msx + wsx) / 2
	scaleY := (msy + wsy) / 2

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
	logDebug("Window destroyed.")
}
