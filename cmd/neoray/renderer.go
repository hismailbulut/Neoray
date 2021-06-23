package main

import (
	"fmt"
)

const (
	UNSUPPORTED_GLYPH_ID    = "Unsupported"
	UNDERCURL_GLYPH_ID      = "Undercurl"
	FONT_ATLAS_DEFAULT_SIZE = 512
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
	userFont    Font
	defaultFont Font
	fontSize    float32
	fontAtlas   FontAtlas
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

type VertexDataStorage struct {
	renderer   *Renderer
	begin, end int
}

func CreateRenderer() Renderer {
	defer measure_execution_time("CreateRenderer")()

	RGL_Init()
	renderer := Renderer{
		fontSize: DEFAULT_FONT_SIZE,
		fontAtlas: FontAtlas{
			texture:    CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE),
			characters: make(map[string]IntRect),
		},
	}

	renderer.defaultFont = CreateDefaultFont()
	EditorSingleton.cellWidth, EditorSingleton.cellHeight = renderer.defaultFont.CalculateCellSize()
	EditorSingleton.calculateCellCount()

	RGL_CreateViewport(EditorSingleton.window.width, EditorSingleton.window.height)
	RGL_SetAtlasTexture(&renderer.fontAtlas.texture)

	return renderer
}

func (renderer *Renderer) SetFont(font Font) {
	renderer.userFont = font
	// reset atlas
	renderer.ClearAtlas()
	// resize default fonts
	renderer.defaultFont.Resize(renderer.userFont.size)
	renderer.fontSize = renderer.userFont.size
	// update cell size if font size has changed
	renderer.UpdateCellSize(&font)
	// redraw all cells
	EditorSingleton.grid.MakeAllCellsChanged()
	renderer.drawCall = true
}

func (renderer *Renderer) SetFontSize(size float32) {
	if size >= MINIMUM_FONT_SIZE {
		renderer.defaultFont.Resize(size)
		if renderer.userFont.size > 0 {
			renderer.userFont.Resize(size)
			renderer.UpdateCellSize(&renderer.userFont)
		} else {
			renderer.UpdateCellSize(&renderer.defaultFont)
		}
		renderer.ClearAtlas()
		EditorSingleton.grid.MakeAllCellsChanged()
		renderer.drawCall = true
		renderer.fontSize = size
		EditorSingleton.nvim.Echo("Font Size: %.1f\n", size)
	}
}

func (renderer *Renderer) IncreaseFontSize() {
	renderer.SetFontSize(renderer.fontSize + 0.5)
}

func (renderer *Renderer) DecreaseFontSize() {
	renderer.SetFontSize(renderer.fontSize - 0.5)
}

func (renderer *Renderer) UpdateCellSize(font *Font) bool {
	w, h := font.CalculateCellSize()
	// Only resize if font metrics are different
	if w != EditorSingleton.cellWidth || h != EditorSingleton.cellHeight {
		EditorSingleton.cellWidth = w
		EditorSingleton.cellHeight = h
		EditorSingleton.nvim.RequestResize()
		return true
	}
	return false
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
			vBegin := cellVertexPos(y, x)
			// prepare vertex buffer
			for i, pos := range positions {
				renderer.vertexData[vBegin+i].pos = pos
			}
			// prepare element buffer
			eBegin := cellElementPos(y, x)
			for i, e := range IndexDataOrder {
				renderer.indexData[eBegin+i] = uint32(vBegin) + e
			}
		}
	}
	// Add cursor to data.
	EditorSingleton.cursor.CreateVertexData()
	// Add popup menu to data.
	EditorSingleton.popupMenu.CreateVertexData()
	// DEBUG: draw font atlas to top right
	if isDebugBuild() {
		renderer.DebugDrawFontAtlas()
	}
	// Update element buffer
	RGL_UpdateIndices(renderer.indexData)
}

func (renderer *Renderer) DebugDrawFontAtlas() {
	atlas_pos := IntRect{
		X: EditorSingleton.window.width - FONT_ATLAS_DEFAULT_SIZE,
		Y: 0,
		W: FONT_ATLAS_DEFAULT_SIZE,
		H: FONT_ATLAS_DEFAULT_SIZE,
	}
	positions := triangulateRect(atlas_pos)
	texture_positions := triangulateFRect(F32Rect{X: 0, Y: 0, W: 1, H: 1})
	renderer.AppendRectData(positions, texture_positions, F32Color{R: 1, G: 1, B: 1, A: 1}, F32Color{})
}

func (storage VertexDataStorage) SetCellPos(index int, pos IntRect) {
	cBegin := storage.begin + (index * 4)
	assert(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	positions := triangulateRect(pos)
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].pos = positions[i]
	}
}

func (storage VertexDataStorage) SetCellTexPos(index int, texPos IntRect) {
	cBegin := storage.begin + (index * 4)
	assert(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	texPositions := triangulateFRect(storage.renderer.fontAtlas.texture.GetRectGLCoordinates(texPos))
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].tex = texPositions[i]
	}
}

func (storage VertexDataStorage) SetCellColor(index int, fg, bg U8Color) {
	cBegin := storage.begin + (index * 4)
	assert(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	fgc := fg.ToF32Color()
	bgc := bg.ToF32Color()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].fg = fgc
		storage.renderer.vertexData[cBegin+i].bg = bgc
	}
}

func (storage VertexDataStorage) SetCellSpColor(index int, sp U8Color) {
	cBegin := storage.begin + (index * 4)
	assert(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	spc := sp.ToF32Color()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].sp = spc
	}
}

func (storage VertexDataStorage) SetAllCellsColors(fg, bg U8Color) {
	fgc := fg.ToF32Color()
	bgc := bg.ToF32Color()
	for i := storage.begin; i < storage.end; i++ {
		storage.renderer.vertexData[i].fg = fgc
		storage.renderer.vertexData[i].bg = bgc
	}
}

// Reserve calculates needed vertex size for given cell count,
// allocates data for it and returns beginning of the index of the reserved data.
// You can set this data using SetVertex* functions. Functions takes index arguments
// as cell positions, not vertex data positions.
func (renderer *Renderer) ReserveVertexData(cellCount int) VertexDataStorage {
	begin := len(renderer.vertexData)
	for i := 0; i < cellCount; i++ {
		renderer.AppendRectData([4]F32Vec2{}, [4]F32Vec2{}, F32Color{}, F32Color{})
	}
	return VertexDataStorage{
		renderer: renderer,
		begin:    begin,
		end:      len(renderer.vertexData),
	}
}

func (renderer *Renderer) AppendRectData(positions [4]F32Vec2, texPositions [4]F32Vec2, fg, bg F32Color) {
	begin := len(renderer.vertexData)
	for i := 0; i < 4; i++ {
		renderer.vertexData = append(renderer.vertexData, Vertex{
			pos: positions[i],
			tex: texPositions[i],
			fg:  fg,
			bg:  bg,
		})
	}
	for _, e := range IndexDataOrder {
		renderer.indexData = append(renderer.indexData, uint32(begin)+e)
	}
}

// This function copies src to dst from left to right,
// and used for scroll acceleration
func (renderer *Renderer) CopyRowData(dst, src, left, right int) {
	defer measure_execution_time("Renderer.CopyRowData")()
	dst_begin := cellVertexPos(dst, left)
	src_begin := cellVertexPos(src, left)
	src_end := cellVertexPos(src, right)
	for i := 0; i < src_end-src_begin; i++ {
		dst_data := &renderer.vertexData[dst_begin+i]
		src_data := renderer.vertexData[src_begin+i]
		dst_data.tex = src_data.tex
		dst_data.fg = src_data.fg
		dst_data.bg = src_data.bg
		dst_data.sp = src_data.sp
	}
}

func (renderer *Renderer) SetCellBgColor(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].bg = c
	}
}

func (renderer *Renderer) SetCellFgColor(x, y int, src IntRect, dest IntRect, color U8Color) {
	c := color.ToF32Color()
	area := renderer.fontAtlas.texture.GetRectGLCoordinates(src)
	texture_coords := triangulateFRect(area)
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex = texture_coords[i]
		renderer.vertexData[begin+i].fg = c
	}
}

func (renderer *Renderer) SetCellSpColor(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].sp = c
	}
}

func (renderer *Renderer) ClearCellFgColor(x, y int) {
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex = F32Vec2{}
		renderer.vertexData[begin+i].fg = F32Color{}
		renderer.vertexData[begin+i].sp = F32Color{}
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

func cellVertexPos(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 4
}

func cellElementPos(x, y int) int {
	return (x*EditorSingleton.columnCount + y) * 6
}

func (renderer *Renderer) NextAtlasPosition(width int) IntVec2 {
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

func (renderer *Renderer) GetSupportedFace(char string, italic, bold bool) (*FontFace, bool) {
	// First try the user font for this character.
	if renderer.userFont.size > 0 {
		face := renderer.userFont.GetSuitableFace(italic, bold)
		if face.IsDrawable(char) {
			return face, true
		}
	}
	// If this is not regular font, try with default non regular fonts.
	if italic || bold {
		face := renderer.defaultFont.GetSuitableFace(italic, bold)
		if face.IsDrawable(char) {
			return face, true
		}
	}
	// Use default font if user font not supports this glyph.
	// Default regular font has more glyphs.
	face := renderer.defaultFont.GetSuitableFace(false, false)
	return face, face.IsDrawable(char)
}

func (renderer *Renderer) CheckUndercurlPos() {
	if _, ok := renderer.fontAtlas.characters[UNDERCURL_GLYPH_ID]; ok == false {
		// Create and update needed texture
		textImage := renderer.defaultFont.regular.renderUndercurl()
		textPos := renderer.NextAtlasPosition(EditorSingleton.cellWidth)
		rect := IntRect{
			X: textPos.X,
			Y: textPos.Y,
			W: EditorSingleton.cellWidth,
			H: EditorSingleton.cellHeight,
		}
		// Draw text to empty position of atlas texture
		renderer.fontAtlas.texture.UpdatePartFromImage(textImage, rect)
		// Add this font to character list for further use
		renderer.fontAtlas.characters[UNDERCURL_GLYPH_ID] = rect
		// Set uniform
		RGL_SetUndercurlRect(renderer.fontAtlas.texture.GetRectGLCoordinates(rect))
	}
}

// Returns given character position at the font atlas.
func (renderer *Renderer) GetCharPos(char string, italic, bold, underline, strikethrough bool) IntRect {
	// generate specific id for this character
	id := fmt.Sprintf("%s%t%t%t%t", char, italic, bold, underline, strikethrough)
	if pos, ok := renderer.fontAtlas.characters[id]; ok == true {
		// use stored texture
		return pos
	} else {
		// Get suitable font and check for glyph
		fontFace, ok := renderer.GetSupportedFace(char, italic, bold)
		if !ok {
			// log_debug("Unsupported glyph:", char, []rune(char))
			id = UNSUPPORTED_GLYPH_ID
			pos, ok := renderer.fontAtlas.characters[id]
			if ok {
				return pos
			}
		}
		// Render character to an image
		textImage := fontFace.RenderChar(char, underline, strikethrough)
		if textImage == nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_RENDERER, "Failed to render glyph", char, []rune(char))
			id = UNSUPPORTED_GLYPH_ID
			pos, ok := renderer.fontAtlas.characters[id]
			if ok {
				return pos
			}
		}
		// Get empty atlas position for this character
		text_pos := renderer.NextAtlasPosition(EditorSingleton.cellWidth)
		position := IntRect{
			X: text_pos.X,
			Y: text_pos.Y,
			W: EditorSingleton.cellWidth,
			H: EditorSingleton.cellHeight,
		}
		// Draw text to empty position of atlas texture
		renderer.fontAtlas.texture.UpdatePartFromImage(textImage, position)
		// Add this font to character list for further use
		renderer.fontAtlas.characters[id] = position
		return position
	}
}

func (renderer *Renderer) DrawCellCustom(x, y int, char string,
	fg, bg, sp U8Color,
	italic, bold, underline, undercurl, strikethrough bool) {
	// draw Background
	renderer.SetCellBgColor(x, y, bg)
	if char == "" || char == " " {
		// this is an empty cell, clear the text vertex data
		renderer.ClearCellFgColor(x, y)
		return
	}
	if undercurl {
		renderer.CheckUndercurlPos()
		renderer.SetCellSpColor(x, y, sp)
	} else {
		renderer.SetCellSpColor(x, y, U8Color{})
	}
	// get character position in atlas texture
	atlasPos := renderer.GetCharPos(char, italic, bold, underline, strikethrough)
	// draw
	renderer.SetCellFgColor(x, y, atlasPos, cellRect(x, y), fg)
}

func (renderer *Renderer) DrawCellWithAttrib(x, y int, cell Cell, attrib HighlightAttribute) {
	// attrib id 0 is default palette
	fg := EditorSingleton.grid.default_fg
	bg := EditorSingleton.grid.default_bg
	sp := EditorSingleton.grid.default_sp
	// bg transparency, this only affects default attribute backgrounds
	bg.A = EditorSingleton.backgroundAlpha()
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
	// Draw cell
	renderer.DrawCellCustom(x, y, cell.char, fg, bg, sp,
		attrib.italic, attrib.bold, attrib.underline, attrib.undercurl, attrib.strikethrough)
}

func (renderer *Renderer) DrawCell(x, y int, cell Cell) {
	if cell.attrib_id > 0 {
		renderer.DrawCellWithAttrib(x, y, cell, EditorSingleton.grid.attributes[cell.attrib_id])
	} else {
		// transparency
		bg := EditorSingleton.grid.default_bg
		bg.A = EditorSingleton.backgroundAlpha()
		renderer.DrawCellCustom(x, y, cell.char,
			EditorSingleton.grid.default_fg, bg, EditorSingleton.grid.default_sp,
			false, false, false, false, false)
	}
}

// Don't call this function directly. Set drawCall value to true in the renderer.
// This function sets renderCall to true and draws cursor one more time.
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

// Don't call this function directly. Set renderCall value to true in the renderer.
func (renderer *Renderer) Render() {
	RGL_ClearScreen(EditorSingleton.grid.default_bg)
	RGL_UpdateVertices(renderer.vertexData)
	RGL_Render()
	EditorSingleton.window.handle.SwapBuffers()
}

func (renderer *Renderer) Close() {
	renderer.fontAtlas.texture.Delete()
	RGL_Close()
}
