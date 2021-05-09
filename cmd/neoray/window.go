package main

import (
	"log"

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

	sdl_window, err := sdl.CreateWindow(
		title,
		sdl.WINDOWPOS_CENTERED,
		sdl.WINDOWPOS_CENTERED,
		int32(width), int32(height),
		sdl.WINDOW_RESIZABLE)

	if err != nil {
		log.Fatalln(err)
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
	}
}

func (window *Window) Update(editor *Editor) {
	window.HandleWindowResizing(editor)
	HandleNvimRedrawEvents(editor)
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