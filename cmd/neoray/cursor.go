package main

import (
	rl "github.com/chunqian/go-raylib/raylib"
)

type Cursor struct {
	X       int
	Y       int
	lastRow int
	lastCol int
	animPos rl.Vector2
}

func (cursor *Cursor) SetPosition(x, y int) {
	cursor.lastRow = cursor.Y
	cursor.lastCol = cursor.X
	cursor.animPos = rl.Vector2{
		X: float32(cursor.lastRow), Y: float32(cursor.lastCol),
	}
	cursor.X = x
	cursor.Y = y
}

func (cursor *Cursor) Draw(grid *Grid, canvas *Canvas, mode *Mode) {

	cell := &grid.cells[cursor.X][cursor.Y]

	mode_info := mode.mode_infos[mode.current_mode_name]

	// initialize swapped
	fg := grid.default_bg
	bg := grid.default_fg

	sp := grid.default_sp

	italic := false
	bold := false

	if mode_info.attr_id != 0 {
		attrib := grid.attributes[mode_info.attr_id]
		fg = attrib.foreground
		bg = attrib.background
		sp = attrib.special
		// font
		italic = attrib.italic
		bold = attrib.bold
		// reverse color if reverse attribute set
		if attrib.reverse {
			fg, bg = bg, fg
		}
		if attrib.underline || attrib.undercurl {
			fg = sp
		}
	}

	cell_pos := rl.Vector2{
		X: canvas.cell_width * float32(cursor.Y),
		Y: canvas.cell_height * float32(cursor.X),
	}

	var rect rl.Rectangle
	draw_char := false

	switch mode_info.cursor_shape {
	case "block":
		rect = rl.Rectangle{
			X:      cell_pos.X,
			Y:      cell_pos.Y,
			Width:  canvas.cell_width,
			Height: canvas.cell_height,
		}
		draw_char = true
		break
	case "horizontal":
		height := canvas.cell_height * (float32(mode_info.cell_percentage) / 100)
		rect = rl.Rectangle{
			X:      cell_pos.X,
			Y:      cell_pos.Y + (canvas.cell_height - height),
			Width:  canvas.cell_width,
			Height: height,
		}
		break
	case "vertical":
		rect = rl.Rectangle{
			X:      cell_pos.X,
			Y:      cell_pos.Y,
			Width:  canvas.cell_width * (float32(mode_info.cell_percentage) / 100),
			Height: canvas.cell_height,
		}
		break
	}

	rl.DrawRectangleRec(rect, bg)

	if draw_char {
		char_pos := rl.Vector2{
			X: cell_pos.X,
			Y: cell_pos.Y + (canvas.cell_width / 3),
		}
		rl.DrawTextEx(canvas.font.GetDrawableFont(italic, bold),
			cell.char, char_pos, canvas.font.size, 0, fg)
	}
}
