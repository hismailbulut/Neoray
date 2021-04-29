package main

import (
	rl "github.com/chunqian/go-raylib/raylib"
)

const (
	CURSOR_SHAPE_BLOCK = iota
	CURSOR_SHAPE_HORIZONTAL
	CURSOR_SHAPE_VERTICAL
)

type Cursor struct {
	X               int
	Y               int
	shape           int
	cell_percentage int
	blinkwait       int
	blinkon         bool
	blinkoff        bool
	attr_id         int
	attr_id_lm      int
	short_name      string
	name            string
}

func (cursor *Cursor) Draw(w *Window) {
	// When attr_id is 0, the background and foreground
	// colors should be swapped.
	fg := w.table.default_fg
	// bg := w.table.default_fg

	if cursor.attr_id > 0 {
		attr := &w.table.attributes[cursor.attr_id-1]
		if attr.foreground != rl.Black {
			fg = attr.foreground
		}
		// if attr.background != rl.Black {
		//     bg = attr.background
		// }
	}

	rect := rl.Rectangle{
		X:      float32(cursor.Y) * w.canvas.cell_width,
		Y:      float32(cursor.X) * w.canvas.cell_height,
		Width:  w.canvas.cell_width,
		Height: w.canvas.cell_height,
	}

	fg.A = 175

	rl.DrawRectangleRec(rect, fg) // background
}
