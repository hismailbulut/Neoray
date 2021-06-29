package main

import (
	"fmt"
)

const (
	UNSUPPORTED_GLYPH_ID = "Unsupported"
	UNDERCURL_GLYPH_ID   = "Undercurl"
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
	// If this is true, the DrawAllChangedCells function will be called from Update.
	drawCall bool
}

type VertexDataStorage struct {
	renderer   *Renderer
	begin, end int
}

func CreateRenderer() Renderer {
	defer measure_execution_time()()

	rglInit()
	renderer := Renderer{
		fontSize: DEFAULT_FONT_SIZE,
		fontAtlas: FontAtlas{
			texture:    CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE),
			characters: make(map[string]IntRect),
		},
	}

	renderer.defaultFont = CreateDefaultFont()
	EditorSingleton.cellWidth, EditorSingleton.cellHeight = renderer.defaultFont.GetCellSize()
	EditorSingleton.calculateCellCount()

	rglCreateViewport(EditorSingleton.window.width, EditorSingleton.window.height)
	rglSetAtlasTexture(&renderer.fontAtlas.texture)

	return renderer
}

func (renderer *Renderer) setFont(font Font) {
	renderer.userFont = font
	// resize default fonts
	renderer.defaultFont.Resize(font.size)
	// update cell size if font size has changed
	renderer.updateCellSize(&font)
	// reset atlas
	renderer.clearAtlas()
	renderer.fontSize = font.size
}

func (renderer *Renderer) setFontSize(size float32) {
	if size >= MINIMUM_FONT_SIZE && size != renderer.fontSize {
		renderer.defaultFont.Resize(size)
		if renderer.userFont.size > 0 {
			renderer.userFont.Resize(size)
			renderer.updateCellSize(&renderer.userFont)
		} else {
			renderer.updateCellSize(&renderer.defaultFont)
		}
		renderer.clearAtlas()
		renderer.fontSize = size
		EditorSingleton.nvim.echoMsg("Font Size: %.1f", size)
	}
}

func (renderer *Renderer) increaseFontSize() {
	renderer.setFontSize(renderer.fontSize + 0.5)
}

func (renderer *Renderer) decreaseFontSize() {
	renderer.setFontSize(renderer.fontSize - 0.5)
}

func (renderer *Renderer) updateCellSize(font *Font) bool {
	w, h := font.GetCellSize()
	// Only resize if font metrics are different
	if w != EditorSingleton.cellWidth || h != EditorSingleton.cellHeight {
		EditorSingleton.cellWidth = w
		EditorSingleton.cellHeight = h
		EditorSingleton.nvim.requestResize()
		return true
	}
	return false
}

// This function will only be called from neovim.
func (renderer *Renderer) resize(rows, cols int) {
	renderer.createVertexData(rows, cols)
	rglCreateViewport(EditorSingleton.window.width, EditorSingleton.window.height)
}

func (renderer *Renderer) clearAtlas() {
	renderer.fontAtlas.texture.Clear()
	renderer.fontAtlas.characters = make(map[string]IntRect)
	renderer.fontAtlas.pos = IntVec2{}
	EditorSingleton.grid.makeAllCellsChanged()
	EditorSingleton.popupMenu.updateChars()
	renderer.drawCall = true
}

func (renderer *Renderer) createVertexData(rows, cols int) {
	defer measure_execution_time()()
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
	EditorSingleton.cursor.createVertexData()
	// Add popup menu to data.
	EditorSingleton.popupMenu.createVertexData()
	// DEBUG: draw font atlas to top right
	if isDebugBuild() {
		renderer.debugDrawFontAtlas()
	}
	// Update element buffer
	rglUpdateIndices(renderer.indexData)
}

func (renderer *Renderer) debugDrawFontAtlas() {
	atlas_pos := IntRect{
		X: EditorSingleton.window.width - FONT_ATLAS_DEFAULT_SIZE,
		Y: 0,
		W: FONT_ATLAS_DEFAULT_SIZE,
		H: FONT_ATLAS_DEFAULT_SIZE,
	}
	positions := triangulateRect(atlas_pos)
	texture_positions := triangulateFRect(F32Rect{X: 0, Y: 0, W: 1, H: 1})
	renderer.appendRectData(positions, texture_positions, F32Color{R: 1, G: 1, B: 1, A: 1}, F32Color{})
}

func (storage VertexDataStorage) setCellPos(index int, pos IntRect) {
	cBegin := storage.begin + (index * 4)
	assert(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	positions := triangulateRect(pos)
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].pos = positions[i]
	}
}

func (storage VertexDataStorage) setCellTexPos(index int, texPos IntRect) {
	cBegin := storage.begin + (index * 4)
	assert(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	texPositions := triangulateFRect(storage.renderer.fontAtlas.texture.GetRectGLCoordinates(texPos))
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].tex = texPositions[i]
	}
}

func (storage VertexDataStorage) setCellColor(index int, fg, bg U8Color) {
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

func (storage VertexDataStorage) setCellSpColor(index int, sp U8Color) {
	cBegin := storage.begin + (index * 4)
	assert(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	spc := sp.ToF32Color()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].sp = spc
	}
}

func (storage VertexDataStorage) setAllCellsColors(fg, bg U8Color) {
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
func (renderer *Renderer) reserveVertexData(cellCount int) VertexDataStorage {
	begin := len(renderer.vertexData)
	for i := 0; i < cellCount; i++ {
		renderer.appendRectData([4]F32Vec2{}, [4]F32Vec2{}, F32Color{}, F32Color{})
	}
	return VertexDataStorage{
		renderer: renderer,
		begin:    begin,
		end:      len(renderer.vertexData),
	}
}

func (renderer *Renderer) appendRectData(positions [4]F32Vec2, texPositions [4]F32Vec2, fg, bg F32Color) {
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
func (renderer *Renderer) copyRowData(dst, src, left, right int) {
	defer measure_execution_time()()
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

func (renderer *Renderer) setCellBgColor(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].bg = c
	}
}

func (renderer *Renderer) setCellFgColor(x, y int, src IntRect, dest IntRect, color U8Color) {
	c := color.ToF32Color()
	area := renderer.fontAtlas.texture.GetRectGLCoordinates(src)
	texture_coords := triangulateFRect(area)
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex = texture_coords[i]
		renderer.vertexData[begin+i].fg = c
	}
}

func (renderer *Renderer) setCellSpColor(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].sp = c
	}
}

func (renderer *Renderer) clearCellFgColor(x, y int) {
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex = F32Vec2{}
		renderer.vertexData[begin+i].fg = F32Color{}
		renderer.vertexData[begin+i].sp = F32Color{}
	}
}

func (renderer *Renderer) getCellData(x, y int) [4]Vertex {
	begin := cellVertexPos(x, y)
	return [4]Vertex{
		renderer.vertexData[begin+0],
		renderer.vertexData[begin+1],
		renderer.vertexData[begin+2],
		renderer.vertexData[begin+3],
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

func (renderer *Renderer) nextAtlasPosition(width int) IntVec2 {
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

func (renderer *Renderer) getSupportedFace(char rune, italic, bold bool) (*FontFace, bool) {
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

func (renderer *Renderer) checkUndercurlPos() {
	if _, ok := renderer.fontAtlas.characters[UNDERCURL_GLYPH_ID]; ok == false {
		// Create and update needed texture
		textImage := renderer.defaultFont.regular.renderUndercurl()
		textPos := renderer.nextAtlasPosition(EditorSingleton.cellWidth)
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
		rglSetUndercurlRect(renderer.fontAtlas.texture.GetRectGLCoordinates(rect))
	}
}

// Returns given character position at the font atlas.
func (renderer *Renderer) getCharPos(char rune, italic, bold, underline, strikethrough bool) IntRect {
	dassert(char != ' ', "char is space")
	dassert(char != 0, "char is zero")
	// generate specific id for this character
	id := fmt.Sprintf("%d%t%t%t%t", char, italic, bold, underline, strikethrough)
	if pos, ok := renderer.fontAtlas.characters[id]; ok == true {
		// use stored texture
		return pos
	} else {
		// Get suitable font and check for glyph
		fontFace, ok := renderer.getSupportedFace(char, italic, bold)
		if !ok {
			// If this character can't be drawed, an empty rectangle will be drawed.
			// And we are reducing this rectangle count in the font atlas to 1.
			// Every unsupported glyph will use it.
			// TODO: sprintf is for debugging, delete it
			id = fmt.Sprint(UNSUPPORTED_GLYPH_ID, char)
			pos, ok := renderer.fontAtlas.characters[id]
			if ok {
				return pos
			}
			log_debug("Unsupported glyph:", string(char), char)
		}
		// Render character to an image
		textImage := fontFace.RenderChar(char, underline, strikethrough)
		if textImage == nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_RENDERER, "Failed to render glyph:", string(char), char)
			id = UNSUPPORTED_GLYPH_ID
			pos, ok := renderer.fontAtlas.characters[id]
			if ok {
				return pos
			}
		}
		// TODO: This is always equal to cellWidth for now. But we need
		// to implement some functionality to render double width characters.
		width := textImage.Rect.Dx()
		// Get empty atlas position for this character
		text_pos := renderer.nextAtlasPosition(width)
		position := IntRect{
			X: text_pos.X,
			Y: text_pos.Y,
			W: width,
			H: EditorSingleton.cellHeight,
		}
		// Draw text to empty position of atlas texture
		renderer.fontAtlas.texture.UpdatePartFromImage(textImage, position)
		// Add this font to character list for further use
		renderer.fontAtlas.characters[id] = position
		return position
	}
}

func (renderer *Renderer) DrawCellCustom(x, y int, char rune,
	fg, bg, sp U8Color,
	italic, bold, underline, undercurl, strikethrough bool) {
	// draw Background
	renderer.setCellBgColor(x, y, bg)
	if char == 0 {
		// this is an empty cell, clear the text vertex data
		renderer.clearCellFgColor(x, y)
		return
	}
	if undercurl {
		renderer.checkUndercurlPos()
		renderer.setCellSpColor(x, y, sp)
	} else {
		renderer.setCellSpColor(x, y, U8Color{})
	}
	// get character position in atlas texture
	atlasPos := renderer.getCharPos(char, italic, bold, underline, strikethrough)
	// draw
	renderer.setCellFgColor(x, y, atlasPos, cellRect(x, y), fg)
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
	if cell.attribId > 0 {
		renderer.DrawCellWithAttrib(x, y, cell, EditorSingleton.grid.attributes[cell.attribId])
	} else {
		// transparency
		bg := EditorSingleton.grid.default_bg
		bg.A = EditorSingleton.backgroundAlpha()
		renderer.DrawCellCustom(x, y, cell.char,
			EditorSingleton.grid.default_fg, bg, EditorSingleton.grid.default_sp,
			false, false, false, false, false)
	}
}

func (renderer *Renderer) Update() {
	if renderer.drawCall {
		renderer.drawAllChangedCells()
		renderer.drawCall = false
	}
	if renderer.renderCall {
		renderer.render()
		renderer.renderCall = false
	}
}

// Don't call this function directly. Set drawCall value to true in the renderer.
// This function sets renderCall to true and draws cursor one more time.
func (renderer *Renderer) drawAllChangedCells() {
	defer measure_execution_time()()
	// Set changed cells vertex data
	total := 0
	for x, row := range EditorSingleton.grid.cells {
		for y, cell := range row {
			if cell.needsDraw {
				renderer.DrawCell(x, y, cell)
				total++
				EditorSingleton.grid.cells[x][y].needsDraw = false
			}
		}
	}
	if total > 0 {
		// Draw cursor one more time.
		EditorSingleton.cursor.Draw()
	}
	// Render changes
	renderer.renderCall = true
}

// Don't call this function directly. Set renderCall value to true in the renderer.
func (renderer *Renderer) render() {
	rglClearScreen(EditorSingleton.grid.default_bg)
	rglUpdateVertices(renderer.vertexData)
	rglRender()
	EditorSingleton.window.handle.SwapBuffers()
}

func (renderer *Renderer) Close() {
	renderer.fontAtlas.texture.Delete()
	rglClose()
}
