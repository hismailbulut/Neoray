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
			characters: make(map[string]sdl.Rect),
		},
		cell_width:  cell_width,
		cell_height: cell_height,
	}

	RGL_Init(window)
	RGL_CreateViewport(window.width, window.height)

	renderer.font_atlas.texture = CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE)
	RGL_SetAtlasTexture(&renderer.font_atlas.texture)

	renderer.Resize(window.width, window.height)

	return renderer
}

func (renderer *Renderer) Resize(w, h int) {
	renderer.window_width = w
	renderer.window_height = h
	col_count := renderer.window_width / renderer.cell_width
	row_count := renderer.window_height / renderer.cell_height
	renderer.col_count = col_count
	renderer.row_count = row_count
	renderer.CreateVertexData()
	RGL_CreateViewport(w, h)
}

func (renderer *Renderer) CreateVertexData() {
	// Stride is the size of cells multiplied by 6 because
	// every cell has 2 triangles and every triangle has 6 vertices.
	renderer.vertex_data_stride = renderer.col_count * renderer.row_count * 6
	// First area is for background drawing
	// Second area is for text drawing
	// Last 6 vertex is for cursor
	renderer.vertex_data_size = (2 * renderer.vertex_data_stride) + 6
	renderer.vertex_data = make([]Vertex, renderer.vertex_data_size, renderer.vertex_data_size)
	for y := 0; y < renderer.row_count; y++ {
		for x := 0; x < renderer.col_count; x++ {
			// prepare first area
			cell_rect := renderer.GetCellRect(y, x)
			positions := triangulate_rect(&cell_rect)
			begin1 := renderer.GetCellVertexPosition(y, x)
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
	// DEBUG: draw font atlas to top right
	renderer.DebugDrawFontAtlas()
}

func (renderer *Renderer) DebugDrawFontAtlas() {
	atlas_pos := sdl.Rect{
		X: int32(renderer.window_width - int(FONT_ATLAS_DEFAULT_SIZE)),
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

func (renderer *Renderer) SetCellTextData(x, y int, src sdl.Rect, dest sdl.Rect, color sdl.Color) {
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

func (renderer *Renderer) ClearCellTextData(x, y int) {
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

func (renderer *Renderer) SetCursorRectData(rect sdl.Rect, color sdl.Color) {
	c := u8color_to_fcolor(color)
	positions := triangulate_rect(&rect)
	begin := renderer.vertex_data_stride * 2
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

func (renderer *Renderer) GetCellVertexPosition(x, y int) int {
	return (x*renderer.col_count + y) * 6
}

func (renderer *Renderer) GetEmptyAtlasPosition(width int) ivec2 {
	atlas := &renderer.font_atlas
	pos := atlas.pos
	atlas.pos.X += width
	if atlas.pos.X+width > int(FONT_ATLAS_DEFAULT_SIZE) {
		atlas.pos.X = 0
		atlas.pos.Y += renderer.cell_height
	}
	if atlas.pos.Y+renderer.cell_height > int(FONT_ATLAS_DEFAULT_SIZE) {
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
		renderer.ClearCellTextData(x, y)
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

func (renderer *Renderer) DrawCellWithAttrib(x, y int, cell Cell, attrib HighlightAttribute, fg, bg, sp sdl.Color) {
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

func (renderer *Renderer) DrawCell(x, y int, cell Cell, grid *Grid) {
	if cell.attrib_id > 0 {
		renderer.DrawCellWithAttrib(x, y, cell, grid.attributes[cell.attrib_id],
			grid.default_fg, grid.default_bg, grid.default_sp)
	} else {
		renderer.DrawCellCustom(x, y, cell.char, grid.default_fg, grid.default_bg, false, false)
	}
}

// TODO: Cursor cells are not redrawing after scroll.
func (renderer *Renderer) Draw(editor *Editor) {
	defer measure_execution_time("Render.Draw")()
	RGL_ClearScreen(editor.grid.default_bg)

	redrawed_rows := 0
	redrawed_cells := 0
	for x, row := range editor.grid.cells {
		if editor.grid.changed_rows[x] == true {
			for y, cell := range row {
				if cell.changed {
					renderer.DrawCell(x, y, cell, &editor.grid)
					editor.grid.cells[x][y].changed = false
					redrawed_cells++
				}
			}
			editor.grid.changed_rows[x] = false
			redrawed_rows++
		}
	}
	log_debug_msg("Redrawed Cells:", redrawed_cells, "Rows:", redrawed_rows)

	// Draw cursor
	editor.cursor.Draw(editor)
	// Render changes
	RGL_Render(renderer.vertex_data)
	// Swap window surface
	editor.window.handle.GLSwap()
}

func (renderer *Renderer) Close() {
	renderer.font_atlas.texture.Delete()
	renderer.font.Unload()
	RGL_Close()
}
