package main

import (
	rl "github.com/chunqian/go-raylib/raylib"
)

type Canvas struct {
	texture     rl.RenderTexture2D
	font        Font
	cell_width  float32
	cell_height float32
}

func (canvas *Canvas) LoadTexture(width, height int32) {
	canvas.texture = rl.LoadRenderTexture(width, height)
	// rl.GenTextureMipmaps((*rl.Texture2D)(&c.texture.Texture))
	// rl.SetTextureFilter((rl.Texture2D)(c.texture.Texture), int32(rl.FILTER_TRILINEAR))
}

func (canvas *Canvas) UnloadTexture() {
	rl.UnloadRenderTexture(canvas.texture)
}

func (canvas *Canvas) GetTexture() rl.Texture2D {
	return rl.Texture2D(canvas.texture.Texture)
}

func (canvas *Canvas) DrawCell(grid *Grid, cell *Cell, mode *Mode, pos rl.Vector2, last_column_in_row bool) {

	rect := rl.Rectangle{
		X:      pos.X,
		Y:      pos.Y,
		Width:  canvas.cell_width,
		Height: canvas.cell_height,
	}

	fg := grid.default_fg
	bg := grid.default_bg
	sp := grid.default_sp

	italic := false
	bold := false

	if cell.attrib_id > 0 {
		// set attribute colors
		attrib := grid.attributes[cell.attrib_id]
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

	// background
	rl.DrawRectangleRec(rect, bg)
	if last_column_in_row {
		rl.DrawRectangleRec(
			rl.Rectangle{X: rect.X + rect.Width, Y: rect.Y,
				Width: rect.Width, Height: rect.Height},
			bg)
	}

	char_pos := rl.Vector2{
		X: rect.X,
		Y: rect.Y + (rect.Width / 3),
	}

	rl.DrawTextEx(canvas.font.GetDrawableFont(italic, bold),
		cell.char, char_pos, canvas.font.size, 0, fg)
}

func (canvas *Canvas) Draw(grid *Grid, mode *Mode, cursor *Cursor) {
	rl.BeginTextureMode(canvas.texture)

	for x := 0; x < len(grid.cells); x++ {
		// only draw if this row changed
		if grid.changed_rows[x] == true {

			for y := 0; y < len(grid.cells[x]); y++ {
				// cell position
				pos := rl.Vector2{
					X: canvas.cell_width * float32(y),
					Y: canvas.cell_height * float32(x)}
				// draw this cell
				canvas.DrawCell(grid,
					&grid.cells[x][y],
					mode,
					pos,
					y == len(grid.cells[x])-1)
			}

			grid.changed_rows[x] = false
		}
	}

	cursor.Draw(grid, canvas, mode)
	grid.changed_rows[cursor.X] = true

	rl.EndTextureMode()
}
