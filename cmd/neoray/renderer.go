package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	FONT_ATLAS_DEFAULT_SIZE int = 512
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
	font        Font
	defaultFont Font
	fontAtlas   FontAtlas
	fontLoaded  bool
	// Vertex data holds vertices for cells. Every cell has 4 vertice and every vertice
	// holds position, atlas texture position, foreground and background color for cell.
	// Positions doesn't change excepts cursor and only updating when initializing and resizing.
	vertexData []Vertex
	// Index data holds indices for cells. Every cell has 6 indices. Index data is created
	// and updated only when initializing and resizing.
	indexData []uint32
	// If this is true, the Render function will be called from Update.
	renderCall bool
	drawCall   bool
}

func CreateRenderer() Renderer {
	RGL_Init()
	FontSystemInit()
	renderer := Renderer{
		fontAtlas: FontAtlas{
			texture:    CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE),
			characters: make(map[string]sdl.Rect),
		},
	}
	renderer.defaultFont, _ = CreateFont("", DEFAULT_FONT_SIZE)
	EditorSingleton.cellWidth, EditorSingleton.cellHeight = renderer.defaultFont.CalculateCellSize()
	CalculateCellCount()
	RGL_CreateViewport(EditorSingleton.window.width, EditorSingleton.window.height)
	RGL_SetAtlasTexture(&renderer.fontAtlas.texture)
	return renderer
}

func (renderer *Renderer) SetFont(font Font) {
	if renderer.font.size == 0 && !renderer.fontLoaded {
		// first time
		renderer.font = font
		renderer.fontLoaded = true
	} else {
		// reloading
		renderer.font.Unload()
		renderer.font = font
		// reset atlas
		renderer.fontAtlas.texture.Clear()
		renderer.fontAtlas.characters = make(map[string]sdl.Rect)
		renderer.fontAtlas.pos = ivec2{}
		// Redraw every cell.
	}
	// We need to check if the given font is valid,
	// otherwise caller wants to disable user font and use
	// system font and we will use defaultFont
	var cellWidth, cellHeight int
	if font.size > 0 {
		cellWidth, cellHeight = font.CalculateCellSize()
	} else {
		cellWidth, cellHeight = renderer.defaultFont.CalculateCellSize()
	}
	// Only resize if font metrics is different
	if cellWidth != EditorSingleton.cellWidth || cellHeight != EditorSingleton.cellHeight {
		// We need to resize because cell dimensions has been changed
		EditorSingleton.cellWidth = cellWidth
		EditorSingleton.cellHeight = cellHeight
		EditorSingleton.nvim.RequestResize()
	}
	EditorSingleton.grid.MakeAllCellsChanged()
	renderer.drawCall = true
}

func CalculateCellCount() {
	EditorSingleton.columnCount = EditorSingleton.window.width / EditorSingleton.cellWidth
	EditorSingleton.rowCount = EditorSingleton.window.height / EditorSingleton.cellHeight
	EditorSingleton.cellCount = EditorSingleton.columnCount * EditorSingleton.rowCount
}

// Call CalculateCellCount before this function.
func (renderer *Renderer) Resize(rows, cols int) {
	renderer.CreateVertexData(rows, cols)
	RGL_CreateViewport(EditorSingleton.window.width, EditorSingleton.window.height)
}

func (renderer *Renderer) CreateVertexData(rows, cols int) {
	defer measure_execution_time("Renderer.CreateVertexData")()
	cellCount := rows * cols
	vertex_data_size := 4 * (cellCount + 1)
	renderer.vertexData = make([]Vertex, vertex_data_size, vertex_data_size)
	index_data_size := 6 * (cellCount + 1)
	renderer.indexData = make([]uint32, index_data_size, index_data_size)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			cell_rect := cellRect(y, x)
			positions := triangulateRect(&cell_rect)
			vBegin := cellVertexPosition(y, x)
			// prepare vertex buffer
			for i, pos := range positions {
				renderer.vertexData[vBegin+i].pos = pos
			}
			// prepare element buffer
			eBegin := cellElementPosition(y, x)
			for i, e := range IndexDataOrder {
				renderer.indexData[eBegin+i] = uint32(vBegin) + e
			}
		}
	}
	// prepare cursor element buffer
	vBegin := 4 * cellCount
	eBegin := 6 * cellCount
	for i, e := range IndexDataOrder {
		renderer.indexData[eBegin+i] = uint32(vBegin) + e
	}
	// DEBUG: draw font atlas to top right
	renderer.DebugDrawFontAtlas()
	// Update element buffer
	RGL_UpdateElementData(renderer.indexData)
}

func (renderer *Renderer) DebugDrawFontAtlas() {
	atlas_pos := sdl.Rect{
		X: int32(EditorSingleton.window.width - int(FONT_ATLAS_DEFAULT_SIZE)),
		Y: 0,
		W: int32(FONT_ATLAS_DEFAULT_SIZE),
		H: int32(FONT_ATLAS_DEFAULT_SIZE),
	}
	vertex := Vertex{fg: f32color{R: 1, G: 1, B: 1, A: 1}}
	positions := triangulateRect(&atlas_pos)
	texture_positions := triangulateFRect(&sdl.FRect{X: 0, Y: 0, W: 1, H: 1})
	for i := 0; i < 4; i++ {
		vertex.pos = positions[i]
		vertex.tex = texture_positions[i]
		renderer.vertexData = append(renderer.vertexData, vertex)
	}
	vBegin := len(renderer.vertexData) - 4
	for _, e := range IndexDataOrder {
		renderer.indexData = append(renderer.indexData, uint32(vBegin)+e)
	}
}

// This function copies src to dst from left to right,
// and used for scroll acceleration
func (renderer *Renderer) CopyRowData(dst, src, left, right int) {
	defer measure_execution_time("Renderer.CopyRowData")()
	dst_begin := cellVertexPosition(dst, left)
	src_begin := cellVertexPosition(src, left)
	src_end := cellVertexPosition(src, right)
	for i := 0; i < src_end-src_begin; i++ {
		dst_data := &renderer.vertexData[dst_begin+i]
		src_data := renderer.vertexData[src_begin+i]
		dst_data.tex = src_data.tex
		dst_data.fg = src_data.fg
		dst_data.bg = src_data.bg
	}
}

func (renderer *Renderer) SetCellBackgroundData(x, y int, color sdl.Color) {
	c := color_u8_to_f32(color)
	begin := cellVertexPosition(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].bg = c
	}
}

func (renderer *Renderer) SetCellForegroundData(x, y int, src sdl.Rect, dest sdl.Rect, color sdl.Color) {
	c := color_u8_to_f32(color)
	area := renderer.fontAtlas.texture.GetRectGLCoordinates(&src)
	texture_coords := triangulateFRect(&area)
	begin := cellVertexPosition(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex = texture_coords[i]
		renderer.vertexData[begin+i].fg = c
	}
}

func (renderer *Renderer) ClearCellForegroundData(x, y int) {
	begin := cellVertexPosition(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex.X = 0
		renderer.vertexData[begin+i].tex.Y = 0
		renderer.vertexData[begin+i].fg.R = 0
		renderer.vertexData[begin+i].fg.G = 0
		renderer.vertexData[begin+i].fg.B = 0
		renderer.vertexData[begin+i].fg.A = 0
	}
}

func (renderer *Renderer) SetCursorData(pos sdl.Rect, atlas_pos sdl.Rect, fg, bg sdl.Color) {
	bgc := color_u8_to_f32(bg)
	fgc := color_u8_to_f32(fg)
	positions := triangulateRect(&pos)
	atlas_pos_gl := renderer.fontAtlas.texture.GetRectGLCoordinates(&atlas_pos)
	texture_positions := triangulateFRect(&atlas_pos_gl)
	begin := 4 * EditorSingleton.cellCount
	for i := 0; i < 4; i++ {
		// background
		renderer.vertexData[begin+i].pos = positions[i]
		renderer.vertexData[begin+i].tex = texture_positions[i]
		renderer.vertexData[begin+i].fg = fgc
		renderer.vertexData[begin+i].bg = bgc
	}
}

func cellRect(x, y int) sdl.Rect {
	return sdl.Rect{
		X: int32(y * EditorSingleton.cellWidth),
		Y: int32(x * EditorSingleton.cellHeight),
		W: int32(EditorSingleton.cellWidth),
		H: int32(EditorSingleton.cellHeight),
	}
}

func cellVertexPosition(x, y int) int {
	r := (x*EditorSingleton.columnCount + y) * 4
	if r >= len(EditorSingleton.renderer.vertexData) {
		log_debug_msg("BREAK")
	}
	return r
}

func cellElementPosition(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 6
}

func (renderer *Renderer) GetEmptyAtlasPosition(width int) ivec2 {
	atlas := &renderer.fontAtlas
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
	if pos, ok := renderer.fontAtlas.characters[id]; ok == true {
		// use stored texture
		position = pos
	} else {
		// Get suitable font
		var font_handle *ttf.Font
		if renderer.font.size > 0 {
			font_handle = renderer.font.GetSuitableFont(italic, bold)
		} else {
			font_handle = renderer.defaultFont.GetSuitableFont(italic, bold)
		}
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
			// log_debug_msg("Char:", char, "Width:", position.W, "Height:", position.H, "DEFAULT:",
			//     EditorSingleton.cellWidth, EditorSingleton.cellHeight)
		}
		// Draw text to empty position of atlas texture
		renderer.fontAtlas.texture.UpdatePartFromSurface(text_surface, &position)
		// Add this font to character list for further use
		renderer.fontAtlas.characters[id] = position
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
	renderer.SetCellForegroundData(x, y, atlas_char_pos, cellRect(x, y), fg)
}

func (renderer *Renderer) DrawCellWithAttrib(x, y int, cell Cell, attrib HighlightAttribute) {
	// attrib id 0 is default palette
	fg := EditorSingleton.grid.default_fg
	bg := EditorSingleton.grid.default_bg
	sp := EditorSingleton.grid.default_sp
	// set attribute colors
	if !colorIsBlack(attrib.foreground) {
		fg = attrib.foreground
	}
	if !colorIsBlack(attrib.background) {
		bg = attrib.background
	}
	if !colorIsBlack(attrib.special) {
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
	// Set changed cells vertex data
	rendered := 0
	for x, row := range EditorSingleton.grid.cells {
		for y, cell := range row {
			if cell.needs_redraw {
				renderer.DrawCell(x, y, cell)
				rendered++
				EditorSingleton.grid.cells[x][y].needs_redraw = false
			}
		}
	}
	if rendered > 0 {
		// Draw cursor one more time.
		EditorSingleton.cursor.Draw()
	}
	// Render changes
	renderer.renderCall = true
}

func (renderer *Renderer) Update() {
	if renderer.drawCall {
		renderer.DrawAllChangedCells()
		renderer.drawCall = false
	}
	if renderer.renderCall {
		renderer.Render()
		renderer.renderCall = false
	}
}

// Don't call this function directly. Call SetRender instead.
func (renderer *Renderer) Render() {
	RGL_ClearScreen(EditorSingleton.grid.default_bg)
	RGL_UpdateVertexData(renderer.vertexData)
	RGL_Render()
	EditorSingleton.window.handle.GLSwap()
}

func (renderer *Renderer) Close() {
	renderer.fontAtlas.texture.Delete()
	renderer.font.Unload()
	renderer.defaultFont.Unload()
	FontSystemClose()
	RGL_Close()
}
