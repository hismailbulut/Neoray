package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

type UIOptions struct {
	arabicshape   bool
	ambiwidth     string
	emoji         bool
	guifont       string
	guifontset    string
	guifontwide   string
	linespace     int
	pumblend      int
	showtabline   int
	termguicolors bool
}

type Window struct {
	handle *sdl.Window
	width  int
	height int
	title  string
}

func CreateWindow(width int, height int, title string) Window {
	window := Window{
		width:  width,
		height: height,
		title:  title,
	}

	sdl_window, err := sdl.CreateWindow(title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED,
		int32(width), int32(height), sdl.WINDOW_RESIZABLE|sdl.WINDOW_OPENGL)
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL window:", err)
	}
	window.handle = sdl_window

	return window
}

func (window *Window) HandleWindowResizing(editor *Editor) {
	w, h := window.handle.GetSize()
	if w != int32(window.width) || h != int32(window.height) {
		window.width = int(w)
		window.height = int(h)
		editor.nvim.ResizeUI(editor)
		editor.renderer.Resize(int(w), int(h))
	}
}

func (window *Window) Update(editor *Editor) {
	window.HandleWindowResizing(editor)
	HandleNvimRedrawEvents(editor)
	// DEBUG
	fps_string := fmt.Sprintf("FPS: %d", frames_per_second)
	window.SetTitle(fps_string)
}

func (window *Window) SetSize(newWidth int, newHeight int, editor *Editor) {
	window.handle.SetSize(int32(newWidth), int32(newHeight))
	window.HandleWindowResizing(editor)
}

func (window *Window) SetTitle(title string) {
	window.handle.SetTitle(title)
	window.title = title
}

func (window *Window) Close() {
	window.handle.Destroy()
}
