package main

import (
	"fmt"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

type Window struct {
	handle     *sdl.Window
	title      string
	width      int
	height     int
	fullscreen bool
}

func CreateWindow(width int, height int, title string) Window {
	window := Window{
		title:      title,
		width:      width,
		height:     height,
		fullscreen: false,
	}

	sdl_window, err := sdl.CreateWindow(title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		int32(width), int32(height), sdl.WINDOW_OPENGL|sdl.WINDOW_RESIZABLE)
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL window:", err)
	}
	window.handle = sdl_window

	sdl.GLSetAttribute(sdl.GL_CONTEXT_PROFILE_MASK, sdl.GL_CONTEXT_PROFILE_CORE)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 3)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 3)

	return window
}

func (window *Window) HandleWindowResizing() {
	w, h := window.handle.GetSize()
	if w != int32(window.width) || h != int32(window.height) {
		window.width = int(w)
		window.height = int(h)
		CalculateCellCount()
		EditorSingleton.nvim.ResizeUI()
		EditorSingleton.renderer.Resize()
	}
}

func (window *Window) Update() {
	window.HandleWindowResizing()
	// DEBUG
	fps_string := fmt.Sprintf(" | FPS: %d", EditorSingleton.framesPerSecond)
	idx := strings.LastIndex(window.title, " | ")
	if idx == -1 {
		window.SetTitle(window.title + fps_string)
	} else {
		window.SetTitle(window.title[0:idx] + fps_string)
	}
}

func (window *Window) SetSize(newWidth int, newHeight int, editor *Editor) {
	window.handle.SetSize(int32(newWidth), int32(newHeight))
	window.HandleWindowResizing()
}

func (window *Window) SetTitle(title string) {
	window.handle.SetTitle(title)
	window.title = title
}

func (window *Window) ToggleFullscreen() {
	if window.fullscreen {
		window.handle.SetFullscreen(0)
		window.fullscreen = false
	} else {
		window.handle.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
		window.fullscreen = true
	}
}

func (window *Window) Close() {
	window.handle.Destroy()
}
