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

	WINDOW_SIZE_AUTO = 1 << 31
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
	rW := videoMode.Width
	rH := videoMode.Height

	if width == WINDOW_SIZE_AUTO {
		width = (rW / 5) * 3
	}
	if height == WINDOW_SIZE_AUTO {
		height = (rH / 4) * 3
	}

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

	// NOTE: When doublebuffering is on, framebuffer transparency not working on fullscreen
	glfw.WindowHint(glfw.DoubleBuffer, glfw.False)

	windowHandle, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to create glfw window:", err)
	}
	window.handle = windowHandle

	window.handle.SetFramebufferSizeCallback(windowResizeHandler)
	window.handle.SetIconifyCallback(windowMinimizeHandler)

	window.calculateDPI()

	window.handle.MakeContextCurrent()

	return window
}

func windowResizeHandler(w *glfw.Window, width, height int) {
	EditorSingleton.window.width = width
	EditorSingleton.window.height = height
	if width > 0 && height > 0 {
		EditorSingleton.nvim.requestResize()
	}
}

func windowMinimizeHandler(w *glfw.Window, minimized bool) {
	EditorSingleton.window.minimized = minimized
}

func (window *Window) Update() {
	if isDebugBuild() {
		fps_string := fmt.Sprintf(" | TPS: %d | Delta: %f",
			EditorSingleton.updatesPerSecond, EditorSingleton.deltaTime)
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
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Unknown window state:", state)
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
	pWidth, pHeight := monitor.GetPhysicalSize()
	pDiagonal := math.Sqrt(float64(pWidth*pWidth) + float64(pHeight*pHeight))
	pDiagonalInch := pDiagonal * 0.0393700787
	mWidth := float32(monitor.GetVideoMode().Width)
	mHeight := float32(monitor.GetVideoMode().Height)
	mDiagonal := math.Sqrt(float64(mWidth*mWidth) + float64(mHeight*mHeight))
	window.dpi = mDiagonal / pDiagonalInch
	log_debug("Monitor diagonal:", pDiagonalInch, "dpi:", window.dpi)
}

func (window *Window) Close() {
	window.handle.Destroy()
}
