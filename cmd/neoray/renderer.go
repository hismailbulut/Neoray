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
	characters map[string]sdl.Rect
}

type Renderer struct {
	font               Font
	font_atlas         FontAtlas
	vertex_data        []Vertex
	vertex_data_size   int
	vertex_data_stride int
}

func CreateRenderer(font Font) Renderer {
	renderer := Renderer{
		font: font,
		font_atlas: FontAtlas{
			characters: make(map[string]sdl.Rect),
		},
	}

	EditorSingleton.cellWidth, EditorSingleton.cellHeight = font.CalculateCellSize()

	EditorSingleton.columnCount = EditorSingleton.window.width / EditorSingleton.cellWidth
	EditorSingleton.rowCount = EditorSingleton.window.height / EditorSingleton.cellHeight

	RGL_Init()

	renderer.font_atlas.texture = CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE)
	RGL_SetAtlasTexture(&renderer.font_atlas.texture)

	renderer.Resize()
	return renderer
}

func (renderer *Renderer) Resize() {
	renderer.CreateVertexData()
	RGL_CreateViewport(EditorSingleton.window.width, EditorSingleton.window.height)
}

func (renderer *Renderer) CreateVertexData() {
	// Stride is the size of cells multiplied by 6 because
	// every cell has 2 triangles and every triangle has 6 vertices.
	renderer.vertex_data_stride = EditorSingleton.columnCount * EditorSingleton.rowCount * 6
	// First area is for background drawing
	// Second area is for text drawing
	// Last 12 vertex is for cursor
	renderer.vertex_data_size = 2*renderer.vertex_data_stride + 12
	renderer.vertex_data = make([]Vertex, renderer.vertex_data_size, renderer.vertex_data_size)
	for y := 0; y < EditorSingleton.rowCount; y++ {
		for x := 0; x < EditorSingleton.columnCount; x++ {
			// prepare first area
			cell_rect := renderer.GetCellRect(y, x)
			positions := triangulate_rect(&cell_rect)
			begin := renderer.GetCellVertexPosition(y, x)
			for i, pos := range positions {
				renderer.vertex_data[begin+i].X = float32(pos.X)
				renderer.vertex_data[begin+i].Y = float32(pos.Y)
			}
			// prepare second area
			begin += renderer.vertex_data_stride
			for i, pos := range positions {
				renderer.vertex_data[begin+i].X = float32(pos.X)
				renderer.vertex_data[begin+i].Y = float32(pos.Y)
				renderer.vertex_data[begin+i].useTexture = 1
			}
		}
	}
	// DEBUG: draw font atlas to top right
	renderer.DebugDrawFontAtlas()
}

func (renderer *Renderer) DebugDrawFontAtlas() {
	atlas_pos := sdl.Rect{
		X: int32(EditorSingleton.window.width - int(FONT_ATLAS_DEFAULT_SIZE)),
		Y: 0,
		W: int32(FONT_ATLAS_DEFAULT_SIZE),
		H: int32(FONT_ATLAS_DEFAULT_SIZE),
	}
	vertex := Vertex{useTexture: 1, R: 1, G: 1, B: 1, A: 1}
	positions := triangulate_rect(&atlas_pos)
	texture_positions := triangulate_frect(&sdl.FRect{X: 0, Y: 0, W: 1, H: 1})
	for i := 0; i < 6; i++ {
		vertex.X = float32(positions[i].X)
		vertex.Y = float32(positions[i].Y)
		vertex.TexX = texture_positions[i].X
		vertex.TexY = texture_positions[i].Y
		renderer.vertex_data = append(renderer.vertex_data, vertex)
	}
}

// TODO: Find a way to speed up this function.
// This function directly copies one row data to another.
// And used for accelerating scroll operations.
// But still slow.
func (renderer *Renderer) CopyRowData(dst, src, left, right int) {
	defer measure_execution_time("Renderer.CopyRowData")()
	// Move background data first
	dst_begin := renderer.GetCellVertexPosition(dst, left)
	// dst_end := renderer.GetCellVertexPosition(dst, right)
	src_begin := renderer.GetCellVertexPosition(src, left)
	src_end := renderer.GetCellVertexPosition(src, right)
	for i := 0; i < src_end-src_begin; i++ {
		renderer.vertex_data[dst_begin+i].R = renderer.vertex_data[src_begin+i].R
		renderer.vertex_data[dst_begin+i].G = renderer.vertex_data[src_begin+i].G
		renderer.vertex_data[dst_begin+i].B = renderer.vertex_data[src_begin+i].B
		renderer.vertex_data[dst_begin+i].A = renderer.vertex_data[src_begin+i].A
	}
	// Move foreground data
	dst_begin += renderer.vertex_data_stride
	// dst_end += renderer.vertex_data_stride
	src_begin += renderer.vertex_data_stride
	src_end += renderer.vertex_data_stride
	for i := 0; i < src_end-src_begin; i++ {
		renderer.vertex_data[dst_begin+i].TexX = renderer.vertex_data[src_begin+i].TexX
		renderer.vertex_data[dst_begin+i].TexY = renderer.vertex_data[src_begin+i].TexY
		renderer.vertex_data[dst_begin+i].R = renderer.vertex_data[src_begin+i].R
		renderer.vertex_data[dst_begin+i].G = renderer.vertex_data[src_begin+i].G
		renderer.vertex_data[dst_begin+i].B = renderer.vertex_data[src_begin+i].B
		renderer.vertex_data[dst_begin+i].A = renderer.vertex_data[src_begin+i].A
	}
}

func (renderer *Renderer) SetCellBackgroundData(x, y int, color sdl.Color) {
	c := u8color_to_fcolor(color)
	begin := renderer.GetCellVertexPosition(x, y)
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].R = c.R
		renderer.vertex_data[begin+i].G = c.G
		renderer.vertex_data[begin+i].B = c.B
		renderer.vertex_data[begin+i].A = c.A
	}
}

func (renderer *Renderer) SetCellForegroundData(x, y int, src sdl.Rect, dest sdl.Rect, color sdl.Color) {
	area := renderer.font_atlas.texture.GetRectGLCoordinates(&src)
	c := u8color_to_fcolor(color)
	texture_coords := triangulate_frect(&area)
	begin := renderer.vertex_data_stride + renderer.GetCellVertexPosition(x, y)
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].TexX = texture_coords[i].X
		renderer.vertex_data[begin+i].TexY = texture_coords[i].Y
		renderer.vertex_data[begin+i].R = c.R
		renderer.vertex_data[begin+i].G = c.G
		renderer.vertex_data[begin+i].B = c.B
		renderer.vertex_data[begin+i].A = c.A
	}
}

func (renderer *Renderer) ClearCellForegroundData(x, y int) {
	begin := renderer.vertex_data_stride + renderer.GetCellVertexPosition(x, y)
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].TexX = 0
		renderer.vertex_data[begin+i].TexY = 0
		renderer.vertex_data[begin+i].R = 0
		renderer.vertex_data[begin+i].G = 0
		renderer.vertex_data[begin+i].B = 0
		renderer.vertex_data[begin+i].A = 0
	}
}

func (renderer *Renderer) SetCursorData(pos sdl.Rect, atlas_pos sdl.Rect, fg, bg sdl.Color) {
	// Set background data first
	bgc := u8color_to_fcolor(bg)
	positions := triangulate_rect(&pos)
	begin := 2 * renderer.vertex_data_stride
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].X = float32(positions[i].X)
		renderer.vertex_data[begin+i].Y = float32(positions[i].Y)
		renderer.vertex_data[begin+i].R = bgc.R
		renderer.vertex_data[begin+i].G = bgc.G
		renderer.vertex_data[begin+i].B = bgc.B
		renderer.vertex_data[begin+i].A = bgc.A
	}
	// Set foreground data
	begin += 6
	atlas_pos_gl := renderer.font_atlas.texture.GetRectGLCoordinates(&atlas_pos)
	texture_positions := triangulate_frect(&atlas_pos_gl)
	fgc := u8color_to_fcolor(fg)
	for i := 0; i < 6; i++ {
		renderer.vertex_data[begin+i].X = float32(positions[i].X)
		renderer.vertex_data[begin+i].Y = float32(positions[i].Y)
		renderer.vertex_data[begin+i].TexX = texture_positions[i].X
		renderer.vertex_data[begin+i].TexY = texture_positions[i].Y
		renderer.vertex_data[begin+i].useTexture = 1
		renderer.vertex_data[begin+i].R = fgc.R
		renderer.vertex_data[begin+i].G = fgc.G
		renderer.vertex_data[begin+i].B = fgc.B
		renderer.vertex_data[begin+i].A = fgc.A
	}
}

func (renderer *Renderer) GetCellRect(x, y int) sdl.Rect {
	return sdl.Rect{
		X: int32(y * EditorSingleton.cellWidth),
		Y: int32(x * EditorSingleton.cellHeight),
		W: int32(EditorSingleton.cellWidth),
		H: int32(EditorSingleton.cellHeight),
	}
}

func (renderer *Renderer) GetCellVertexPosition(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 6
}

func (renderer *Renderer) GetEmptyAtlasPosition(width int) ivec2 {
	atlas := &renderer.font_atlas
	pos := atlas.pos
	atlas.pos.X += width
	if atlas.pos.X+width > int(FONT_ATLAS_DEFAULT_SIZE) {
		atlas.pos.X = 0
		atlas.pos.Y += EditorSingleton.cellHeight
	}
	if atlas.pos.Y+EditorSingleton.cellHeight > int(FONT_ATLAS_DEFAULT_SIZE) {
		// Fully filled
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_RENDERER, "Font atlas is full.")
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
		position = pos
	} else {
		// Get suitable font
		font_handle := renderer.font.GetSuitableFont(italic, bold)
		// Get text glyph metrics
		// metrics, err := font_handle.GlyphMetrics(rune(char[0]))
		// if err != nil {
		//     log_message(LOG_LEVEL_WARN, LOG_TYPE_RENDERER, "Failed to get glyph metrics of", char, ":", err)
		//     return sdl.Rect{}, err
		// }
		// log_debug_msg("Char:", char, "Metrics:", metrics)
		// Render text to surface
		text_surface, err := font_handle.RenderUTF8Blended(char, COLOR_WHITE)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_RENDERER, "Failed to render text:", err)
			return sdl.Rect{}, err
		}
		defer text_surface.Free()
		// Get empty atlas position for this char
		text_pos := renderer.GetEmptyAtlasPosition(int(text_surface.W))
		position = sdl.Rect{
			X: int32(text_pos.X),
			Y: int32(text_pos.Y),
			W: text_surface.W,
			H: text_surface.H,
		}
		// Draw text to empty position of atlas texture
		renderer.font_atlas.texture.UpdatePartFromSurface(text_surface, &position)
		// Add this font to character list for further use
		renderer.font_atlas.characters[id] = position
	}
	return position, nil
}

func (renderer *Renderer) DrawCellCustom(x, y int, char string, fg, bg sdl.Color, italic, bold bool) {
	// draw Background
	renderer.SetCellBackgroundData(x, y, bg)
	if len(char) == 0 || char == " " {
		// this is an empty cell, clear the text vertex data
		renderer.ClearCellForegroundData(x, y)
		return
	}
	// get character position in atlas texture
	atlas_char_pos, err := renderer.GetCharacterAtlasPosition(char, italic, bold)
	if err != nil {
		return
	}
	// draw
	renderer.SetCellForegroundData(x, y, atlas_char_pos, renderer.GetCellRect(x, y), fg)
}

func (renderer *Renderer) DrawCellWithAttrib(x, y int, cell Cell, attrib HighlightAttribute) {
	// attrib id 0 is default palette
	fg := EditorSingleton.grid.default_fg
	bg := EditorSingleton.grid.default_bg
	sp := EditorSingleton.grid.default_sp
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
	// reverse foreground and background
	if attrib.reverse {
		fg, bg = bg, fg
	}
	// underline and undercurl uses special color for foreground
	if attrib.underline || attrib.undercurl {
		fg = sp
	}
	// Draw cell
	renderer.DrawCellCustom(x, y, cell.char, fg, bg, attrib.italic, attrib.bold)
}

func (renderer *Renderer) DrawCursor() {
	defer measure_execution_time("Renderer.DrawCursor")()
	info := EditorSingleton.cursor.GetDrawInfo()
	cell := EditorSingleton.grid.cells[info.x][info.y]
	if info.draw_char && len(cell.char) != 0 && cell.char != " " {
		// We need to draw cell character to the cursor foreground.
		// Because cursor is not transparent.
		italic := false
		bold := false
		if cell.attrib_id > 0 {
			attrib := EditorSingleton.grid.attributes[cell.attrib_id]
			italic = attrib.italic
			bold = attrib.bold
		}
		atlas_pos, err := renderer.GetCharacterAtlasPosition(cell.char, italic, bold)
		if err != nil {
			return
		}
		renderer.SetCursorData(info.rect, atlas_pos, info.fg, info.bg)
	} else {
		// No cell drawing needed. Just draw the cursor.
		renderer.SetCursorData(info.rect, sdl.Rect{}, sdl.Color{}, info.bg)
	}
	renderer.Render()
}

func (renderer *Renderer) DrawCell(x, y int, cell Cell) {
	if cell.attrib_id > 0 {
		renderer.DrawCellWithAttrib(x, y, cell, EditorSingleton.grid.attributes[cell.attrib_id])
	} else {
		renderer.DrawCellCustom(x, y, cell.char, EditorSingleton.grid.default_fg, EditorSingleton.grid.default_bg, false, false)
	}
}

func (renderer *Renderer) DrawAllChangedCells() {
	defer measure_execution_time("Render.Draw")()
	for x, row := range EditorSingleton.grid.cells {
		for y, cell := range row {
			if cell.needs_redraw {
				renderer.DrawCell(x, y, cell)
				EditorSingleton.grid.cells[x][y].needs_redraw = false
			}
		}
	}
	// Cursor needs redrawing.
	EditorSingleton.cursor.needs_redraw = true
	// Render changes
	renderer.Render()
}

func (renderer *Renderer) Render() {
	defer measure_execution_time("Renderer.Render")()
	RGL_ClearScreen(EditorSingleton.grid.default_bg)
	RGL_Render(renderer.vertex_data)
	EditorSingleton.window.handle.GLSwap()
}

func (renderer *Renderer) Close() {
	renderer.font_atlas.texture.Delete()
	renderer.font.Unload()
	RGL_Close()
}
