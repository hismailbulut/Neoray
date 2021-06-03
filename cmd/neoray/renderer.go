package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	FONT_ATLAS_DEFAULT_SIZE int = 256
)

var (
	// Generate index data using this order.
	IndexDataOrder = [6]uint32{0, 1, 2, 2, 3, 0}
)

type FontAtlas struct {
	texture    Texture
	pos        ivec2
	characters map[string]sdl.Rect
}

type Renderer struct {
	font       Font
	font_atlas FontAtlas
	// Vertex data is a buffer that stores vertices for cells.
	// Every cell has 8 vertice, first 4 is for background and second 4
	// is for foreground. Example is first 8 vertex is for cell 0. And
	// second 8 is for cell 1 and so on. Last 8 vertex is for cursor.
	// Vertex data positions is fixed and initialized in CreateVertexData.
	// Others are changing in runtime continuously. And we are updating
	// vertex data every render call. Total vertex count is 4*2*cell_count+8
	vertex_data []Vertex
	// Index data is a buffer that stores indices for how data will be rendered.
	// The triangle order is defined at IndexDataOrder. Index data will not change
	// on runtime (excepts Resize) and initialized and uploaded at CreateVertexData.
	index_data []uint32
}

func CreateRenderer(font Font) Renderer {
	renderer := Renderer{
		font: font,
		font_atlas: FontAtlas{
			characters: make(map[string]sdl.Rect),
		},
	}
	EditorSingleton.cellWidth, EditorSingleton.cellHeight = font.CalculateCellSize()
	CalculateCellCount()
	RGL_Init()
	renderer.font_atlas.texture = CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE)
	RGL_SetAtlasTexture(&renderer.font_atlas.texture)
	renderer.Resize()
	return renderer
}

func CalculateCellCount() {
	EditorSingleton.columnCount = EditorSingleton.window.width / EditorSingleton.cellWidth
	EditorSingleton.rowCount = EditorSingleton.window.height / EditorSingleton.cellHeight
	EditorSingleton.cellCount = EditorSingleton.columnCount * EditorSingleton.rowCount
}

// Call CalculateCellCount before this function.
func (renderer *Renderer) Resize() {
	renderer.CreateVertexData()
	RGL_CreateViewport(EditorSingleton.window.width, EditorSingleton.window.height)
}

func (renderer *Renderer) CreateVertexData() {
	defer measure_execution_time("Renderer.CreateVertexData")()
	vertex_data_size := 4*2*EditorSingleton.cellCount + 8
	renderer.vertex_data = make([]Vertex, vertex_data_size, vertex_data_size)
	index_data_size := 6*2*EditorSingleton.cellCount + 12
	renderer.index_data = make([]uint32, index_data_size, index_data_size)
	for y := 0; y < EditorSingleton.rowCount; y++ {
		for x := 0; x < EditorSingleton.columnCount; x++ {
			cell_rect := GetCellRect(y, x)
			positions := triangulate_rect(&cell_rect)
			vBegin := GetCellVertexPosition(y, x)
			// prepare vertex buffer
			for i, pos := range positions {
				// prepare background
				renderer.vertex_data[vBegin+i].X = float32(pos.X)
				renderer.vertex_data[vBegin+i].Y = float32(pos.Y)
				// prepare foreground
				renderer.vertex_data[vBegin+i+4].X = float32(pos.X)
				renderer.vertex_data[vBegin+i+4].Y = float32(pos.Y)
				renderer.vertex_data[vBegin+i+4].useTexture = 1
			}
			// prepare element buffer
			eBegin := GetCellElementPosition(y, x)
			for i, e := range IndexDataOrder {
				renderer.index_data[eBegin+i] = uint32(vBegin) + e
				renderer.index_data[eBegin+i+6] = uint32(vBegin) + e + 4
			}
		}
	}
	// prepare cursor vertex buffer
	vBegin := 4 * 2 * EditorSingleton.cellCount
	for i := 4; i < 8; i++ {
		renderer.vertex_data[vBegin+i].useTexture = 1
	}
	// prepare cursor element buffer
	cBegin := 6 * 2 * EditorSingleton.cellCount
	for i, e := range IndexDataOrder {
		renderer.index_data[cBegin+i] = uint32(vBegin) + e
		renderer.index_data[cBegin+i+6] = uint32(vBegin) + e + 4
	}
	// DEBUG: draw font atlas to top right
	renderer.DebugDrawFontAtlas()
	// Update element buffer
	RGL_UpdateElementData(renderer.index_data)
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
	for i := 0; i < 4; i++ {
		vertex.X = float32(positions[i].X)
		vertex.Y = float32(positions[i].Y)
		vertex.TexX = texture_positions[i].X
		vertex.TexY = texture_positions[i].Y
		renderer.vertex_data = append(renderer.vertex_data, vertex)
	}
	vBegin := len(renderer.vertex_data) - 4
	for _, e := range IndexDataOrder {
		renderer.index_data = append(renderer.index_data, uint32(vBegin)+e)
	}
}

func (renderer *Renderer) CopyRowData(dst, src, left, right int) {
	defer measure_execution_time("Renderer.CopyRowData")()
	dst_begin := GetCellVertexPosition(dst, left)
	src_begin := GetCellVertexPosition(src, left)
	src_end := GetCellVertexPosition(src, right)
	for j := 0; j < src_end-src_begin; j += 8 {
		for i := j; i < j+4; i++ {
			// copy background data
			renderer.vertex_data[dst_begin+i].R = renderer.vertex_data[src_begin+i].R
			renderer.vertex_data[dst_begin+i].G = renderer.vertex_data[src_begin+i].G
			renderer.vertex_data[dst_begin+i].B = renderer.vertex_data[src_begin+i].B
			renderer.vertex_data[dst_begin+i].A = renderer.vertex_data[src_begin+i].A
			// copy foreground data
			renderer.vertex_data[dst_begin+i+4].TexX = renderer.vertex_data[src_begin+i+4].TexX
			renderer.vertex_data[dst_begin+i+4].TexY = renderer.vertex_data[src_begin+i+4].TexY
			renderer.vertex_data[dst_begin+i+4].R = renderer.vertex_data[src_begin+i+4].R
			renderer.vertex_data[dst_begin+i+4].G = renderer.vertex_data[src_begin+i+4].G
			renderer.vertex_data[dst_begin+i+4].B = renderer.vertex_data[src_begin+i+4].B
			renderer.vertex_data[dst_begin+i+4].A = renderer.vertex_data[src_begin+i+4].A
		}
	}
}

func (renderer *Renderer) SetCellBackgroundData(x, y int, color sdl.Color) {
	c := u8color_to_fcolor(color)
	begin := GetCellVertexPosition(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertex_data[begin+i].R = c.R
		renderer.vertex_data[begin+i].G = c.G
		renderer.vertex_data[begin+i].B = c.B
		renderer.vertex_data[begin+i].A = c.A
	}
}

func (renderer *Renderer) SetCellForegroundData(x, y int, src sdl.Rect, dest sdl.Rect, color sdl.Color) {
	c := u8color_to_fcolor(color)
	area := renderer.font_atlas.texture.GetRectGLCoordinates(&src)
	texture_coords := triangulate_frect(&area)
	begin := GetCellVertexPosition(x, y)
	for i := 4; i < 8; i++ {
		renderer.vertex_data[begin+i].TexX = texture_coords[i-4].X
		renderer.vertex_data[begin+i].TexY = texture_coords[i-4].Y
		renderer.vertex_data[begin+i].R = c.R
		renderer.vertex_data[begin+i].G = c.G
		renderer.vertex_data[begin+i].B = c.B
		renderer.vertex_data[begin+i].A = c.A
	}
}

func (renderer *Renderer) ClearCellForegroundData(x, y int) {
	begin := GetCellVertexPosition(x, y)
	for i := 4; i < 8; i++ {
		renderer.vertex_data[begin+i].TexX = 0
		renderer.vertex_data[begin+i].TexY = 0
		renderer.vertex_data[begin+i].R = 0
		renderer.vertex_data[begin+i].G = 0
		renderer.vertex_data[begin+i].B = 0
		renderer.vertex_data[begin+i].A = 0
	}
}

func (renderer *Renderer) SetCursorData(pos sdl.Rect, atlas_pos sdl.Rect, fg, bg sdl.Color) {
	bgc := u8color_to_fcolor(bg)
	fgc := u8color_to_fcolor(fg)
	positions := triangulate_rect(&pos)
	atlas_pos_gl := renderer.font_atlas.texture.GetRectGLCoordinates(&atlas_pos)
	texture_positions := triangulate_frect(&atlas_pos_gl)
	begin := 4 * 2 * EditorSingleton.cellCount
	for i := 0; i < 4; i++ {
		// background
		renderer.vertex_data[begin+i].X = float32(positions[i].X)
		renderer.vertex_data[begin+i].Y = float32(positions[i].Y)
		renderer.vertex_data[begin+i].R = bgc.R
		renderer.vertex_data[begin+i].G = bgc.G
		renderer.vertex_data[begin+i].B = bgc.B
		renderer.vertex_data[begin+i].A = bgc.A
		// foreground
		renderer.vertex_data[begin+i+4].X = float32(positions[i].X)
		renderer.vertex_data[begin+i+4].Y = float32(positions[i].Y)
		renderer.vertex_data[begin+i+4].TexX = texture_positions[i].X
		renderer.vertex_data[begin+i+4].TexY = texture_positions[i].Y
		renderer.vertex_data[begin+i+4].R = fgc.R
		renderer.vertex_data[begin+i+4].G = fgc.G
		renderer.vertex_data[begin+i+4].B = fgc.B
		renderer.vertex_data[begin+i+4].A = fgc.A
	}
}

func GetCellRect(x, y int) sdl.Rect {
	return sdl.Rect{
		X: int32(y * EditorSingleton.cellWidth),
		Y: int32(x * EditorSingleton.cellHeight),
		W: int32(EditorSingleton.cellWidth),
		H: int32(EditorSingleton.cellHeight),
	}
}

func GetCellVertexPosition(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 8
}

func GetCellElementPosition(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 12
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
		// TODO: Unsupported font glyphs must targets the same position in atlas.
		// Currently we are drawing unsupported glyph for every font and filling
		// the atlas with weird rectangles. If you don't understand please try
		// rendering some glyphs your font is not supporting.
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
		if text_surface.W != int32(EditorSingleton.cellWidth) || text_surface.H != int32(EditorSingleton.cellHeight) {
			log_debug_msg("Char:", char, "Width:", position.W, "Height:", position.H, "DEFAULT:",
				EditorSingleton.cellWidth, EditorSingleton.cellHeight)
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
	renderer.SetCellForegroundData(x, y, atlas_char_pos, GetCellRect(x, y), fg)
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
	// Draw cursor one more time.
	EditorSingleton.cursor.Draw()
	// Render changes
	renderer.Render()
}

func (renderer *Renderer) Render() {
	defer measure_execution_time("Renderer.Render")()
	RGL_ClearScreen(EditorSingleton.grid.default_bg)
	RGL_UpdateVertexData(renderer.vertex_data)
	RGL_Render()
	EditorSingleton.window.handle.GLSwap()
}

func (renderer *Renderer) Close() {
	renderer.font_atlas.texture.Delete()
	renderer.font.Unload()
	RGL_Close()
}
