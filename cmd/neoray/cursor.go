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

type Cursor struct {
	X       int
	Y       int
	lastRow int
	lastCol int
}

func CreateMode() Mode {
	return Mode{
		mode_infos: make(map[string]ModeInfo),
	}
}

func (cursor *Cursor) SetPosition(x, y int, grid *Grid) {
	grid.changed_rows[cursor.X] = true
	grid.cells[cursor.X][cursor.Y].changed = true
	cursor.lastRow = cursor.Y
	cursor.lastCol = cursor.X
	cursor.X = x
	cursor.Y = y
}

func (cursor *Cursor) Draw(editor *Editor) {
	grid := &editor.grid
	mode := &editor.mode
	renderer := &editor.renderer

	mode_info := mode.mode_infos[mode.current_mode_name]

	// initialize swapped
	fg := grid.default_bg
	bg := grid.default_fg

	if mode_info.attr_id != 0 {
		attrib := grid.attributes[mode_info.attr_id]
		fg = attrib.foreground
		bg = attrib.background
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

	cell := grid.cells[cursor.X][cursor.Y]
	if draw_char {
		// If cursor style is block, hide the cursor and
		// redraw the cell with cursor color.
		// TODO: Dont change cell under the cursor. Change cursor itself,
		// draw the character on the cursor rectangle.
		renderer.SetCursorRectData(cursor_rect, sdl.Color{})
		italic := false
		bold := false
		if cell.attrib_id > 0 {
			attrib := grid.attributes[cell.attrib_id]
			italic = attrib.italic
			bold = attrib.bold
		}
		renderer.DrawCellCustom(cursor.X, cursor.Y, cell.char, fg, bg, italic, bold)
	} else {
		// Draw the default cell, the cursor will be drawn to its front
		renderer.DrawCell(cursor.X, cursor.Y, cell, &editor.grid)
		renderer.SetCursorRectData(cursor_rect, bg)
	}
}
