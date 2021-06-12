package main

import (
	"fmt"
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
	pos        IntVec2
	characters map[string]IntRect
}

type Renderer struct {
	userFont       Font
	userFontLoaded bool
	defaultFont    Font
	fontAtlas      FontAtlas
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
	defer measure_execution_time("CreateRenderer")()
	RGL_Init()
	InitializeFontLoader()
	renderer := Renderer{
		fontAtlas: FontAtlas{
			texture:    CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE),
			characters: make(map[string]IntRect),
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
	if !renderer.userFontLoaded {
		// first time
		renderer.userFont = font
		renderer.userFontLoaded = true
	} else {
		// reloading
		renderer.userFont.Unload()
		renderer.userFont = font
		// reset atlas
		renderer.ClearAtlas()
	}
	// update cell size if font size has changed
	renderer.UpdateCellSize(&font)
	// redraw all cells
	EditorSingleton.grid.MakeAllCellsChanged()
	renderer.drawCall = true
}

func (renderer *Renderer) SetDefaultFont(size int) {
	if size != int(renderer.defaultFont.size) {
		// reload default font
		renderer.defaultFont, _ = CreateFont("", size)
	}
	// disable user font
	renderer.userFont.Unload()
	renderer.userFontLoaded = false
	// clear atlas texture
	renderer.ClearAtlas()
	// update cell size if font size has changed
	renderer.UpdateCellSize(&renderer.defaultFont)
	// redraw all cells
	EditorSingleton.grid.MakeAllCellsChanged()
	renderer.drawCall = true
}

func (renderer *Renderer) UpdateCellSize(font *Font) bool {
	var cellWidth, cellHeight int
	cellWidth, cellHeight = font.CalculateCellSize()
	// Only resize if font metrics are different
	if cellWidth != EditorSingleton.cellWidth || cellHeight != EditorSingleton.cellHeight {
		EditorSingleton.cellWidth = cellWidth
		EditorSingleton.cellHeight = cellHeight
		EditorSingleton.nvim.RequestResize()
		return true
	}
	return false
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

func (renderer *Renderer) ClearAtlas() {
	renderer.fontAtlas.texture.Clear()
	renderer.fontAtlas.characters = make(map[string]IntRect)
	renderer.fontAtlas.pos = IntVec2{}
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
			positions := triangulateRect(cell_rect)
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
	if isDebugBuild() {
		// DEBUG: draw font atlas to top right
		renderer.DebugDrawFontAtlas()
	}
	// Update element buffer
	RGL_UpdateElementData(renderer.indexData)
}

func (renderer *Renderer) DebugDrawFontAtlas() {
	atlas_pos := IntRect{
		X: EditorSingleton.window.width - FONT_ATLAS_DEFAULT_SIZE,
		Y: 0,
		W: FONT_ATLAS_DEFAULT_SIZE,
		H: FONT_ATLAS_DEFAULT_SIZE,
	}
	vertex := Vertex{fg: F32Color{R: 1, G: 1, B: 1, A: 1}}
	positions := triangulateRect(atlas_pos)
	texture_positions := triangulateFRect(F32Rect{X: 0, Y: 0, W: 1, H: 1})
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

func (renderer *Renderer) SetCellBackgroundData(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPosition(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].bg = c
	}
}

func (renderer *Renderer) SetCellForegroundData(x, y int, src IntRect, dest IntRect, color U8Color) {
	c := color.ToF32Color()
	area := renderer.fontAtlas.texture.GetRectGLCoordinates(src)
	texture_coords := triangulateFRect(area)
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

func (renderer *Renderer) SetCursorData(pos IntRect, atlas_pos IntRect, fg, bg U8Color) {
	bgc := bg.ToF32Color()
	fgc := fg.ToF32Color()
	positions := triangulateRect(pos)
	atlas_pos_gl := renderer.fontAtlas.texture.GetRectGLCoordinates(atlas_pos)
	texture_positions := triangulateFRect(atlas_pos_gl)
	begin := 4 * EditorSingleton.cellCount
	for i := 0; i < 4; i++ {
		// background
		renderer.vertexData[begin+i].pos = positions[i]
		renderer.vertexData[begin+i].tex = texture_positions[i]
		renderer.vertexData[begin+i].fg = fgc
		renderer.vertexData[begin+i].bg = bgc
	}
}

func cellRect(x, y int) IntRect {
	return IntRect{
		X: y * EditorSingleton.cellWidth,
		Y: x * EditorSingleton.cellHeight,
		W: EditorSingleton.cellWidth,
		H: EditorSingleton.cellHeight,
	}
}

func cellVertexPosition(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 4
}

func cellElementPosition(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 6
}

func (renderer *Renderer) GetEmptyAtlasPosition(width int) IntVec2 {
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
		atlas.pos = IntVec2{}
	}
	return pos
}

func (renderer *Renderer) GetCharacterAtlasPosition(char string, italic, bold bool) (IntRect, error) {
	var position IntRect
	// generate specific id for this character
	id := fmt.Sprintf("%s%t%t", char, italic, bold)
	if pos, ok := renderer.fontAtlas.characters[id]; ok == true {
		// use stored texture
		position = pos
	} else {
		// Get suitable font
		var fontFace FontFace
		if renderer.userFont.size > 0 {
			fontFace = renderer.userFont.GetSuitableFont(italic, bold)
		} else {
			fontFace = renderer.defaultFont.GetSuitableFont(italic, bold)
		}
		// TODO: Detect unsupported glyphs
		textImage := fontFace.RenderChar(char)
		// Get empty atlas position for this char
		text_pos := renderer.GetEmptyAtlasPosition(EditorSingleton.cellWidth)
		position = IntRect{
			X: text_pos.X,
			Y: text_pos.Y,
			W: EditorSingleton.cellWidth,
			H: EditorSingleton.cellHeight,
		}
		// Draw text to empty position of atlas texture
		renderer.fontAtlas.texture.UpdatePartFromImage(textImage, position)
		// Add this font to character list for further use
		renderer.fontAtlas.characters[id] = position
	}
	return position, nil
}

func (renderer *Renderer) DrawCellCustom(x, y int, char string, fg, bg U8Color, italic, bold bool) {
	// draw Background
	renderer.SetCellBackgroundData(x, y, bg)
	if char == "" || char == " " {
		// this is an empty cell, clear the text vertex data
		renderer.ClearCellForegroundData(x, y)
		return
	}
	// get character position in atlas texture
	atlasPos, err := renderer.GetCharacterAtlasPosition(char, italic, bold)
	if err != nil {
		return
	}
	// draw
	renderer.SetCellForegroundData(x, y, atlasPos, cellRect(x, y), fg)
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
	EditorSingleton.window.handle.SwapBuffers()
}

func (renderer *Renderer) Close() {
	renderer.fontAtlas.texture.Delete()
	RGL_Close()
}
