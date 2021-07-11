package main

import (
	"fmt"
	"unicode"
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
	// Init opengl subsystem first.
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
		// TODO: Make all messages optional and can be disabled from init.vim.
		EditorSingleton.nvim.echoMsg("Font Size: %.1f", size)
	}
}

func (renderer *Renderer) DisableUserFont() {
	if renderer.userFont.size > 0 {
		renderer.userFont.size = 0
		renderer.updateCellSize(&renderer.defaultFont)
		renderer.clearAtlas()
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
	renderer.fontAtlas.texture.clear()
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
			positions := cell_rect.positions()
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
	storage := renderer.reserveVertexData(1)
	storage.setCellPos(0, atlas_pos)
	storage.setCellTex(0, IntRect{0, 0, FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE})
	storage.setCellFg(0, U8Color{R: 255, G: 255, B: 255, A: 255})
}

func (storage VertexDataStorage) setCellPos(index int, pos IntRect) {
	cBegin := storage.begin + (index * 4)
	assert_debug(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	positions := pos.positions()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].pos = positions[i]
	}
}

func (storage VertexDataStorage) setCellTex(index int, texPos IntRect) {
	cBegin := storage.begin + (index * 4)
	assert_debug(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	texPositions := storage.renderer.fontAtlas.texture.glCoords(texPos).positions()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].tex = texPositions[i]
	}
}

func (storage VertexDataStorage) setCellFg(index int, fg U8Color) {
	cBegin := storage.begin + (index * 4)
	assert_debug(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	fgc := fg.ToF32Color()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].fg = fgc
	}
}

func (storage VertexDataStorage) setCellBg(index int, bg U8Color) {
	cBegin := storage.begin + (index * 4)
	assert_debug(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	bgc := bg.ToF32Color()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].bg = bgc
	}
}

func (storage VertexDataStorage) setCellSp(index int, sp U8Color) {
	cBegin := storage.begin + (index * 4)
	assert_debug(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	spc := sp.ToF32Color()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].sp = spc
	}
}

func (storage VertexDataStorage) setCellTex2(index int, texPos IntRect) {
	cBegin := storage.begin + (index * 4)
	assert_debug(cBegin >= storage.begin && cBegin+4 <= storage.end,
		"Trying to modify not owned cell. Index:", index, "Begin:", storage.begin, "End:", storage.end)
	texPositions := storage.renderer.fontAtlas.texture.glCoords(texPos).positions()
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[cBegin+i].tex2 = texPositions[i]
	}
}

// Reserve calculates needed vertex size for given cell count,
// allocates data for it and returns beginning of the index of the reserved data.
// You can set this data using SetVertex* functions. Functions takes index arguments
// as cell positions, not vertex data positions.
func (renderer *Renderer) reserveVertexData(cellCount int) VertexDataStorage {
	begin := len(renderer.vertexData)
	for i := 0; i < cellCount; i++ {
		cbegin := len(renderer.vertexData)
		for i := 0; i < 4; i++ {
			renderer.vertexData = append(renderer.vertexData, Vertex{})
		}
		for _, e := range IndexDataOrder {
			renderer.indexData = append(renderer.indexData, uint32(cbegin)+e)
		}
	}
	return VertexDataStorage{
		renderer: renderer,
		begin:    begin,
		end:      len(renderer.vertexData),
	}
}

// This function copies src to dst from left to right,
// and used for accelerating scroll operations.
func (renderer *Renderer) copyRowData(dst, src, left, right int) {
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
		dst_data.tex2 = src_data.tex2
	}
}

func (renderer *Renderer) setCellTex(x, y int, pos IntRect) {
	coords := renderer.fontAtlas.texture.glCoords(pos).positions()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex = coords[i]
	}
}

func (renderer *Renderer) setCellFg(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].fg = c
	}
}

func (renderer *Renderer) setCellBg(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].bg = c
	}
}

func (renderer *Renderer) setCellSp(x, y int, color U8Color) {
	c := color.ToF32Color()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].sp = c
	}
}

func (renderer *Renderer) setCellTex2(x, y int, src IntRect) {
	coords := renderer.fontAtlas.texture.glCoords(src).positions()
	begin := cellVertexPos(x, y)
	for i := 0; i < 4; i++ {
		renderer.vertexData[begin+i].tex2 = coords[i]
	}
}

func (renderer *Renderer) debugGetCellData(x, y int) [4]Vertex {
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
	if pos.X+width >= FONT_ATLAS_DEFAULT_SIZE {
		pos.X = 0
		pos.Y += EditorSingleton.cellHeight
	}
	if pos.Y+EditorSingleton.cellHeight >= FONT_ATLAS_DEFAULT_SIZE {
		// Fully filled
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_RENDERER, "Font atlas is full.")
		renderer.clearAtlas()
		pos = IntVec2{}
	}
	atlas.pos.X = pos.X + width
	atlas.pos.Y = pos.Y
	return pos
}

func (renderer *Renderer) getSupportedFace(char rune, italic, bold bool) (*FontFace, bool) {
	// First try the user font for this character.
	if renderer.userFont.size > 0 {
		face := renderer.userFont.GetSuitableFace(italic, bold)
		if face.ContainsGlyph(char) {
			return face, true
		}
	}
	// If this is not regular font, try with default non regular fonts.
	if italic || bold {
		face := renderer.defaultFont.GetSuitableFace(italic, bold)
		if face.ContainsGlyph(char) {
			return face, true
		}
	}
	// Use default font if user font not supports this glyph.
	// Default regular font has (needs to) more glyphs.
	face := renderer.defaultFont.GetSuitableFace(false, false)
	return face, face.ContainsGlyph(char)
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
		renderer.fontAtlas.texture.updatePart(textImage, rect)
		// Add this font to character list for further use
		renderer.fontAtlas.characters[UNDERCURL_GLYPH_ID] = rect
		// Set uniform
		rglSetUndercurlRect(renderer.fontAtlas.texture.glCoords(rect))
	}
}

// Returns given character position at the font atlas.
func (renderer *Renderer) getCharPos(char rune, italic, bold, underline, strikethrough bool) IntRect {
	assert_debug(char != ' ' && char != 0, "char is zero or space")
	// disable underline or strikethrough if this glyph is not alphanumeric
	if !unicode.IsLetter(char) {
		underline = false
		strikethrough = false
	}
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
			id = UNSUPPORTED_GLYPH_ID
			pos, ok := renderer.fontAtlas.characters[id]
			if ok {
				return pos
			}
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
		renderer.fontAtlas.texture.updatePart(textImage, position)
		// Add this font to character list for further use
		renderer.fontAtlas.characters[id] = position
		return position
	}
}

func (renderer *Renderer) DrawCellCustom(x, y int, char rune,
	fg, bg, sp U8Color,
	italic, bold, underline, undercurl, strikethrough bool) {
	// draw Background
	renderer.setCellBg(x, y, bg)
	if char == 0 {
		// This is an empty cell, clear the vertex data
		if y+1 < EditorSingleton.columnCount {
			renderer.setCellTex2(x, y+1, IntRect{})
		}
		renderer.setCellTex(x, y, IntRect{})
		renderer.setCellSp(x, y, U8Color{})
		return
	}
	if undercurl {
		renderer.checkUndercurlPos()
		renderer.setCellSp(x, y, sp)
	} else {
		renderer.setCellSp(x, y, U8Color{})
	}

	// get character position in atlas texture
	atlasPos := renderer.getCharPos(char, italic, bold, underline, strikethrough)
	if atlasPos.W > EditorSingleton.cellWidth {
		// The atlas width will be 2 times more if the char is a multiwidth char
		// and we are dividing atlas to 2. One for current cell and one for next.
		atlasPos.W /= 2
		if y+1 < EditorSingleton.columnCount {
			// Draw the parts more than width to the next cell.
			// NOTE: The more part has the same color with next cell.
			// NOTE: Multiwidth cells causes glyphs to merge. But we don't care.
			secAtlasPos := IntRect{
				X: atlasPos.X + EditorSingleton.cellWidth,
				Y: atlasPos.Y,
				W: EditorSingleton.cellWidth,
				H: EditorSingleton.cellHeight,
			}
			renderer.setCellTex2(x, y+1, secAtlasPos)
			renderer.setCellFg(x, y+1, fg)
		}
	} else {
		// Clear second texture.
		renderer.setCellTex2(x, y+1, IntRect{})
	}
	// draw
	renderer.setCellTex(x, y, atlasPos)
	renderer.setCellFg(x, y, fg)
}

func (renderer *Renderer) DrawCellWithAttrib(x, y int, cell Cell, attrib HighlightAttribute) {
	// attrib id 0 is default palette
	fg := EditorSingleton.grid.defaultFg
	bg := EditorSingleton.grid.defaultBg
	sp := EditorSingleton.grid.defaultSp
	// bg transparency, this only affects default attribute backgrounds
	bg.A = EditorSingleton.backgroundAlpha()
	// set attribute colors
	if attrib.foreground.A > 0 {
		fg = attrib.foreground
	}
	if attrib.background.A > 0 {
		bg = attrib.background
	}
	if attrib.special.A > 0 {
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
		bg := EditorSingleton.grid.defaultBg
		bg.A = EditorSingleton.backgroundAlpha()
		renderer.DrawCellCustom(x, y, cell.char,
			EditorSingleton.grid.defaultFg, bg, EditorSingleton.grid.defaultSp,
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
	rglClearScreen(EditorSingleton.grid.defaultBg)
	rglUpdateVertices(renderer.vertexData)
	rglRender()
	EditorSingleton.window.handle.SwapBuffers()
}

func (renderer *Renderer) Close() {
	renderer.fontAtlas.texture.Delete()
	rglClose()
}
