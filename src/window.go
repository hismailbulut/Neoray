package main

import (
	rl "github.com/chunqian/go-raylib/raylib"
)

type Window struct {
	width  int
	height int
	title  string
	table  GridTable
	cursor Cursor
	canvas Canvas
	input  Input
}

func CreateAndShow(width int, height int, title string, font_name string, font_size float32) Window {
	rl.SetConfigFlags(uint32(rl.FLAG_WINDOW_RESIZABLE)) // | uint32(rl.FLAG_VSYNC_HINT))

	window := Window{
		width:  width,
		height: height,
		title:  title,
	}

	window.table = CreateGridTable()

	window.canvas = Canvas{
		cell_width:  font_size/2 + 1,
		cell_height: font_size + 3,
	}

	window.cursor = Cursor{}

	window.input = Input{
		options: InputOptions{
			hold_delay_begin:        350,
			hold_delay_between_keys: 30,
		},
	}

	rl.InitWindow(int32(window.width), int32(window.height), window.title)
	rl.SetExitKey(0)

	window.canvas.LoadTexture(int32(window.width), int32(window.height))
	window.canvas.LoadFont(font_name, font_size)

	return window
}

func (w *Window) HandleWindowResizing(proc *NvimProcess) {
	w.width = int(rl.GetScreenWidth())
	w.height = int(rl.GetScreenHeight())
	w.canvas.UnloadTexture()
	w.canvas.LoadTexture(int32(w.width), int32(w.height))
}

func (w *Window) Update(proc *NvimProcess) {
	w.input.HandleInputEvents(proc)
	handle_nvim_updates(proc, w)
	if rl.IsWindowResized() {
		w.HandleWindowResizing(proc)
		proc.ResizeUI(w)
	}
}

func (w *Window) Render() {
	rl.BeginDrawing()
	rl.ClearBackground(w.table.default_bg)

	rl.DrawTextureRec(
		w.canvas.GetTexture(),
		rl.Rectangle{
			X: 0, Y: 0,
			Width:  float32(w.width),
			Height: -float32(w.height)},
		rl.Vector2Zero(),
		rl.White)

	w.cursor.Draw(w)

	rl.EndDrawing()
}

func (w *Window) SetSize(newWidth int, newHeight int, proc *NvimProcess) {
	rl.SetWindowSize(int32(newWidth), int32(newHeight))
	w.HandleWindowResizing(proc)
	proc.ResizeUI(w)
}

func (w *Window) SetTitle(title string) {
	w.title = title
	rl.SetWindowTitle(title)
}

func (w *Window) Close() {
	w.canvas.UnloadFont()
	w.canvas.UnloadTexture()
	rl.CloseWindow()
}
