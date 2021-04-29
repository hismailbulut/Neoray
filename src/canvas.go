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

func (c *Canvas) LoadTexture(width, height int32) {
	c.texture = rl.LoadRenderTexture(width, height)
	// rl.GenTextureMipmaps((*rl.Texture2D)(&c.texture.Texture))
	// rl.SetTextureFilter((rl.Texture2D)(c.texture.Texture), int32(rl.FILTER_TRILINEAR))
}

func (c *Canvas) LoadFont(font_name string, size float32) {
	c.font = Font{}
	c.font.Load(font_name, size)
}

func (c *Canvas) UnloadTexture() {
	rl.UnloadRenderTexture(c.texture)
}

func (c *Canvas) UnloadFont() {
	c.font.Unload()
}

func (c *Canvas) GetTexture() rl.Texture2D {
	return rl.Texture2D(c.texture.Texture)
}

func (c *Canvas) DrawCell(table *GridTable, cell *Cell,
	pos rl.Vector2, draw_text bool) {

	rect := rl.Rectangle{
		X:      pos.X,
		Y:      pos.Y,
		Width:  c.cell_width,
		Height: c.cell_height,
	}

	fg := table.default_fg
	bg := table.default_bg
	sp := table.default_sp

	italic := false
	bold := false

	if cell.attrib_id > 0 {
		// set attribute colors
		attrib := &table.attributes[cell.attrib_id-1]
		if attrib.foreground != rl.Black {
			fg = attrib.foreground
		}
		if attrib.background != rl.Black {
			bg = attrib.background
		}
		if attrib.special != rl.Black {
			sp = attrib.special
		}
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

	rl.DrawRectangleRec(rect, bg)

	// for debug
	// rl.DrawRectangleLinesEx(rect, 1, rl.Black)

	char_pos := rl.Vector2{
		X: rect.X,
		Y: rect.Y + (rect.Width / 3),
	}

	if draw_text {
		// for debug
		char := cell.char
		// if char == "" || char == " " {
		//     char = "."
		// }
		rl.DrawTextEx(c.font.GetDrawableFont(italic, bold), char, char_pos, c.font.size, 0, fg)
	}
}

func (c *Canvas) Draw(table *GridTable) {
	rl.BeginTextureMode(c.texture)

	for x := 0; x < len(table.cells); x++ {
		// only draw if this row changed
		if table.changed_rows[x] == true {

			for y := 0; y < len(table.cells[x]); y++ {
				pos := rl.Vector2{
					X: c.cell_width * float32(y),
					Y: c.cell_height * float32(x)}
				// draw this cell
				c.DrawCell(table, &table.cells[x][y], pos, true)
				// if this is the last column on this row
				if y == len(table.cells[x])-1 {
					pos := rl.Vector2{
						X: c.cell_width * float32(y+1),
						Y: c.cell_height * float32(x)}
					// draw last cell background at the end of the column
					c.DrawCell(table, &table.cells[x][y], pos, false)
				}
			}

			table.changed_rows[x] = false
		}
	}

	rl.EndTextureMode()
}
