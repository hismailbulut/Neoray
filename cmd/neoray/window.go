package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type Window struct {
	handle *glfw.Window
	title  string
	width  int
	height int
	// This is for restoring window from fullscreen, dont use them
	windowedRect IntRect
	minimized    bool
	fullscreen   bool
}

func CreateWindow(width int, height int, title string) Window {
	defer measure_execution_time("CreateWindow")()
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

	// NOTE: When doublebuffer is on, framebuffer transparency not working
	// on fullscreen
	glfw.WindowHint(glfw.DoubleBuffer, glfw.False)

	windowHandle, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to create glfw window:", err)
	}
	window.handle = windowHandle

	window.handle.SetSizeCallback(WindowResizeHandler)
	window.handle.SetIconifyCallback(MinimizeHandler)

	window.handle.MakeContextCurrent()

	return window
}

func MinimizeHandler(w *glfw.Window, minimized bool) {
	EditorSingleton.window.minimized = minimized
}

func WindowResizeHandler(w *glfw.Window, width, height int) {
	EditorSingleton.window.width = width
	EditorSingleton.window.height = height
	EditorSingleton.nvim.RequestResize()
}

func (window *Window) Update() {
	if isDebugBuild() {
		fps_string := fmt.Sprintf(" | TPS: %d | Delta: %f",
			EditorSingleton.framesPerSecond, EditorSingleton.deltaTime)
		idx := strings.LastIndex(window.title, " | TPS:")
		if idx == -1 {
			window.SetTitle(window.title + fps_string)
		} else {
			window.SetTitle(window.title[0:idx] + fps_string)
		}
	}
}

func (window *Window) Raise() {
	window.handle.SetAttrib(glfw.Floating, glfw.True)
	if window.minimized {
		window.handle.Restore()
	}
	// window.handle.Focus()
	window.handle.SetAttrib(glfw.Floating, glfw.False)
}

func (window *Window) SetTitle(title string) {
	window.handle.SetTitle(title)
	window.title = title
}

func (window *Window) ToggleFullscreen() {
	if window.handle.GetMonitor() == nil {
		// to fullscreen
		x, y := window.handle.GetPos()
		w, h := window.handle.GetSize()
		window.windowedRect = IntRect{X: x, Y: y, W: w, H: h}
		monitor := glfw.GetPrimaryMonitor()
		videoMode := monitor.GetVideoMode()
		window.handle.SetMonitor(monitor, 0, 0,
			videoMode.Width, videoMode.Height, videoMode.RefreshRate)
	} else {
		// restore
		window.handle.SetMonitor(nil,
			window.windowedRect.X, window.windowedRect.Y,
			window.windowedRect.W, window.windowedRect.H, 0)
	}
}

func (window *Window) Close() {
	// glfw.DetachCurrentContext()
	window.handle.Destroy()
}
