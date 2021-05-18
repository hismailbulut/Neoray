package main

import "github.com/veandco/go-sdl2/sdl"

type ModeInfo struct {
	cursor_shape    string
	cell_percentage int
	blinkwait       int
	blinkon         int
	blinkoff        int
	attr_id         int
	attr_id_lm      int
	short_name      string
	name            string
}

type Mode struct {
	cursor_style_enabled bool
	mode_infos           map[string]ModeInfo
	current_mode_name    string
	current_mode         int
}

// TODO: Cursor animation
type Cursor struct {
	X       int
	Y       int
	lastRow int
	lastCol int
}

func (cursor *Cursor) SetPosition(x, y int) {
	cursor.lastRow = cursor.Y
	cursor.lastCol = cursor.X
	cursor.X = x
	cursor.Y = y
}

func (cursor *Cursor) Draw(grid *Grid, renderer *Renderer, mode *Mode) {
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

	cell_pos := ivec2{
		X: renderer.cell_width * cursor.Y,
		Y: renderer.cell_height * cursor.X,
	}

	var cursor_rect sdl.Rect
	draw_char := false

	switch mode_info.cursor_shape {
	case "block":
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
		draw_char = true
		break
	case "horizontal":
		height := renderer.cell_height / (100 / mode_info.cell_percentage)
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y + (renderer.cell_height - height)),
			W: int32(renderer.cell_width),
			H: int32(height),
		}
		break
	case "vertical":
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y),
			W: int32(renderer.cell_width / (100 / mode_info.cell_percentage)),
			H: int32(renderer.cell_height),
		}
		break
	}

	if draw_char {
		renderer.SetCursorRect(cursor_rect, sdl.Color{R: 0, G: 0, B: 0, A: 0})
		renderer.DrawCell(cursor.X, cursor.Y, fg, bg, cell.char, italic, bold)
	} else {
		renderer.SetCursorRect(cursor_rect, bg)
	}
}
