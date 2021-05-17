package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	FONT_ATLAS_DEFAULT_SIZE int = 256
)

type FontAtlas struct {
	texture    Texture
	pos        ivec2
	characters map[string]ivec2
}

type Renderer struct {
	font               Font
	font_atlas         FontAtlas
	cell_width         int
	cell_height        int
	window_width       int
	window_height      int
	row_count          int
	col_count          int
	vertex_data        []Vertex
	vertex_data_size   int
	vertex_data_stride int
}

func CreateRenderer(window *Window, font Font) Renderer {
	cell_width, cell_height := font.CalculateCellSize()
	renderer := Renderer{
		font: font,
		font_atlas: FontAtlas{
			characters: make(map[string]ivec2),
		},
		cell_width:  cell_width,
		cell_height: cell_height,
	}

	RGL_Init(window)
	RGL_CreateViewport(window.width, window.height)

	renderer.font_atlas.texture = CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE)
	RGL_SetAtlasTexture(&renderer.font_atlas.texture)

	renderer.Resize(window.width, window.height)

	// DEBUG: draw font atlas to top right
	// atlas_pos := sdl.Rect{
	//     X: int32((editor.grid.width * renderer.cell_width) - int(FONT_ATLAS_DEFAULT_SIZE)),
	//     Y: 0,
	//     W: int32(FONT_ATLAS_DEFAULT_SIZE),
	//     H: int32(FONT_ATLAS_DEFAULT_SIZE),
	// }

	return renderer
}

func (renderer *Renderer) Resize(w, h int) {
	// TODO: These must be global
	renderer.window_width = w
	renderer.window_height = h
	RGL_CreateViewport(w, h)
	col_count := renderer.window_width / renderer.cell_width
	row_count := renderer.window_height / renderer.cell_height
	renderer.col_count = col_count
	renderer.row_count = row_count

	// Create vertex data NOTE:
	// First area is for background drawing
	// Second area is for text drawing
	// Last 6 vertex is for cursor
	renderer.vertex_data_stride = col_count * row_count * 6
	renderer.vertex_data_size = (2 * renderer.vertex_data_stride) + 6
	renderer.vertex_data = make([]Vertex, renderer.vertex_data_size, renderer.vertex_data_size)
	for y := 0; y < row_count; y++ {
		for x := 0; x < col_count; x++ {
			// prepare first area
			cell_rect := renderer.GetCellRect(x, y)
			positions := TriangulateRect(&cell_rect)
			begin1 := (y*col_count + x) * 6
			for i, pos := range positions {
				renderer.vertex_data[begin1+i].X = float32(pos.X)
				renderer.vertex_data[begin1+i].Y = float32(pos.Y)
			}
			// prepare second area
			begin2 := renderer.vertex_data_stride + begin1
			for i, pos := range positions {
				renderer.vertex_data[begin2+i].X = float32(pos.X)
				renderer.vertex_data[begin2+i].Y = float32(pos.Y)
				renderer.vertex_data[begin2+i].useTexture = 1
			}
		}
	}
}

func (renderer *Renderer) SetCellBackground(x, y int, color sdl.Color) {
	c := u8color_to_fcolor(color)
	begin := (x*renderer.col_count + y) * 6
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].R = c.R
		renderer.vertex_data[begin+i].G = c.G
		renderer.vertex_data[begin+i].B = c.B
		renderer.vertex_data[begin+i].A = c.A
	}
}

func (renderer *Renderer) SetCellTextData(x, y int, src sdl.Rect, dest sdl.Rect, color sdl.Color) {
	area := renderer.font_atlas.texture.GetInternalArea(&src)
	// Color and texture id is same for this vertices
	c := u8color_to_fcolor(color)
	texture_coords := TriangulateFRect(&area)
	begin := renderer.vertex_data_stride + ((x*renderer.col_count + y) * 6)
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].TexX = texture_coords[i].X
		renderer.vertex_data[begin+i].TexY = texture_coords[i].Y
		renderer.vertex_data[begin+i].R = c.R
		renderer.vertex_data[begin+i].G = c.G
		renderer.vertex_data[begin+i].B = c.B
		renderer.vertex_data[begin+i].A = c.A
	}
}

func (renderer *Renderer) ClearCellTextData() {
	for i := renderer.vertex_data_stride; i < 2*renderer.vertex_data_stride; i++ {
		renderer.vertex_data[i].TexX = 0
		renderer.vertex_data[i].TexY = 0
		renderer.vertex_data[i].R = 0
		renderer.vertex_data[i].G = 0
		renderer.vertex_data[i].B = 0
		renderer.vertex_data[i].A = 0
	}
}

func (renderer *Renderer) SetCursorRect(rect sdl.Rect, color sdl.Color) {
	c := u8color_to_fcolor(color)
	positions := TriangulateRect(&rect)
	begin := renderer.vertex_data_stride * 2
	assert(begin+6 == renderer.vertex_data_size, "Renderer.SetCursorRect")
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].X = float32(positions[i].X)
		renderer.vertex_data[begin+i].Y = float32(positions[i].Y)
		renderer.vertex_data[begin+i].R = c.R
		renderer.vertex_data[begin+i].G = c.G
		renderer.vertex_data[begin+i].B = c.B
		renderer.vertex_data[begin+i].A = c.A
	}
}

func (renderer *Renderer) GetCellRect(x, y int) sdl.Rect {
	return sdl.Rect{
		X: int32(y * renderer.cell_width),
		Y: int32(x * renderer.cell_height),
		W: int32(renderer.cell_width),
		H: int32(renderer.cell_height),
	}
}

func (renderer *Renderer) GetEmptyAtlasPosition() ivec2 {
	atlas := &renderer.font_atlas
	// calculate position
	pos := atlas.pos
	atlas.pos.X += renderer.cell_width
	if atlas.pos.X+renderer.cell_width > int(FONT_ATLAS_DEFAULT_SIZE) {
		atlas.pos.X = 0
		atlas.pos.Y += renderer.cell_height
	}
	if atlas.pos.Y+renderer.cell_height > int(FONT_ATLAS_DEFAULT_SIZE) {
		// Fully filled
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Font atlas is full.")
		atlas.pos = ivec2{}
	}
	return pos
}

func (renderer *Renderer) GetCharacterAtlasPosition(char string, italic, bold bool) (sdl.Rect, error) {
	var position sdl.Rect
	// generate specific id for this character
	id := fmt.Sprintf("%s%t%t", char, italic, bold)
	if pos, ok := renderer.font_atlas.characters[id]; ok == true {
		// use stored texture
		position = sdl.Rect{
			X: int32(pos.X),
			Y: int32(pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
	} else {
		// Create this text
		font_handle := renderer.font.GetDrawableFont(italic, bold)
		text_surface, err := font_handle.RenderUTF8Blended(char, COLOR_WHITE)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return sdl.Rect{}, err
		}
		defer text_surface.Free()
		// Get empty atlas position
		text_pos := renderer.GetEmptyAtlasPosition()
		position = sdl.Rect{
			X: int32(text_pos.X),
			Y: int32(text_pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
		if text_surface.W > int32(renderer.cell_width) || text_surface.H > int32(renderer.cell_height) {
			// TODO: scale surface or scale texture
			position.W = text_surface.W
			position.H = text_surface.H
		}
		// Draw text to empty position of atlas texture
		renderer.font_atlas.texture.UpdatePartFromSurface(text_surface, &position)
		renderer.font_atlas.characters[id] = ivec2{int(position.X), int(position.Y)}
	}
	return position, nil
}

func (renderer *Renderer) DrawCell(x, y int, fg, bg sdl.Color, char string, italic, bold bool) {
	// draw Background
	renderer.SetCellBackground(x, y, bg)
	if len(char) == 0 || char == " " {
		return
	}
	// get character position in atlas texture
	atlas_char_pos, err := renderer.GetCharacterAtlasPosition(char, italic, bold)
	if err != nil {
		return
	}
	// draw
	renderer.SetCellTextData(x, y, atlas_char_pos, renderer.GetCellRect(x, y), fg)
}

func (renderer *Renderer) DrawCellWithAttrib(x, y int, cell Cell,
	attrib HighlightAttributes, default_fg, default_bg, default_sp sdl.Color) {
	fg := default_fg
	bg := default_bg
	sp := default_sp
	italic := false
	bold := false
	// attrib id 0 is default palette
	// set attribute colors
	if !is_color_black(attrib.foreground) {
		fg = attrib.foreground
	}
	if !is_color_black(attrib.background) {
		bg = attrib.background
	}
	if !is_color_black(attrib.special) {
		sp = attrib.special
	}
	// font
	italic = attrib.italic
	bold = attrib.bold
	// reverse foreground and background
	if attrib.reverse {
		fg, bg = bg, fg
	}
	// underline and undercurl uses special color for foreground
	if attrib.underline || attrib.undercurl {
		fg = sp
	}
	// Draw cell
	renderer.DrawCell(x, y, fg, bg, cell.char, italic, bold)
}

func (renderer *Renderer) Draw(editor *Editor) {
	RGL_ClearScreen(editor.grid.default_bg)
	time_measure_func := measure_execution_time("Renderer.Draw.Loop")
	for x, row := range editor.grid.cells {
		for y, cell := range row {
			// NOTE: Cell attribute id 0 is default attribute
			if cell.attrib_id > 0 {
				renderer.DrawCellWithAttrib(
					x, y, cell, editor.grid.attributes[cell.attrib_id],
					editor.grid.default_fg, editor.grid.default_bg, editor.grid.default_sp)
			} else {
				renderer.DrawCell(
					x, y, editor.grid.default_fg, editor.grid.default_bg, cell.char, false, false)
			}
		}
	}
	time_measure_func()
	// Draw cursor
	editor.cursor.Draw(&editor.grid, &editor.renderer, &editor.mode)
	// Render changes
	RGL_Render(renderer.font_atlas.texture, renderer.vertex_data)
	// Swap window surface
	editor.window.handle.GLSwap()
	// Clear cell text data because we dont want to draw a text to must empty cell
	renderer.ClearCellTextData()
}

func (renderer *Renderer) Close() {
	renderer.font_atlas.texture.Delete()
	renderer.font.Unload()
	RGL_Close()
}
