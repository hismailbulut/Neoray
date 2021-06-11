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
	rect IntRect
}

func CreateWindow(width int, height int, title string) Window {
	window := Window{
		title:  title,
		width:  width,
		height: height,
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.Resizable, glfw.True)

	windowHandle, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to create glfw window:", err)
	}
	window.handle = windowHandle

	window.handle.SetSizeCallback(WindowResizeHandler)
	window.handle.MakeContextCurrent()

	return window
}

func WindowResizeHandler(w *glfw.Window, width, height int) {
	EditorSingleton.window.width = width
	EditorSingleton.window.height = height
	EditorSingleton.nvim.RequestResize()
}

func (window *Window) Update() {
	// DEBUG
	fps_string := fmt.Sprintf(" | FPS: %d", EditorSingleton.framesPerSecond)
	idx := strings.LastIndex(window.title, " | ")
	if idx == -1 {
		window.SetTitle(window.title + fps_string)
	} else {
		window.SetTitle(window.title[0:idx] + fps_string)
	}
}

func (window *Window) SetTitle(title string) {
	window.handle.SetTitle(title)
	window.title = title
}

func (window *Window) ToggleFullscreen() {
	if window.handle.GetMonitor() == nil {
		// Set to fullscreen
		x, y := window.handle.GetPos()
		w, h := window.handle.GetSize()
		window.rect.X = x
		window.rect.Y = y
		window.rect.W = w
		window.rect.H = h
		monitor := glfw.GetPrimaryMonitor()
		videoMode := monitor.GetVideoMode()
		window.handle.SetMonitor(monitor, 0, 0,
			videoMode.Width, videoMode.Height, videoMode.RefreshRate)
	} else {
		// restore
		window.handle.SetMonitor(nil,
			window.rect.X, window.rect.Y,
			window.rect.W, window.rect.H, 0)
	}
}

func (window *Window) Close() {
	glfw.DetachCurrentContext()
	window.handle.Destroy()
}
