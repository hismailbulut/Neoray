package main

import (
	"fmt"
	"unicode"
)

const (
	UNSUPPORTED_GLYPH_ID = "Unsupported"
	UNDERCURL_GLYPH_ID   = "Undercurl"
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
	// Vertex data holds vertices for cells. Every cell has 1 vertex.
	vertexData []Vertex
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
	renderer.updateCellSize(&renderer.defaultFont)

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
	defer measure_execution_time()()
	renderer.fontAtlas.texture.clear()
	renderer.fontAtlas.characters = make(map[string]IntRect)
	renderer.fontAtlas.pos = IntVec2{}
	EditorSingleton.popupMenu.updateChars()
	EditorSingleton.grid.makeAllCellsChanged()
	renderer.drawCall = true
}

func (renderer *Renderer) createVertexData(rows, cols int) {
	defer measure_execution_time()()
	cellCount := rows * cols
	renderer.vertexData = make([]Vertex, cellCount, cellCount)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			renderer.vertexData[cellVertexPos(y, x)].pos = cellPos(y, x)
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
}

func (renderer *Renderer) debugDrawFontAtlas() {
	atlas_pos := F32Rect{
		X: float32(EditorSingleton.window.width - FONT_ATLAS_DEFAULT_SIZE),
		Y: 0,
		W: FONT_ATLAS_DEFAULT_SIZE,
		H: FONT_ATLAS_DEFAULT_SIZE,
	}
	storage := renderer.reserveVertexData(1)
	storage.setCellPos(0, atlas_pos)
	storage.setCellTex(0, IntRect{0, 0, FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE})
	storage.setCellFg(0, U8Color{R: 255, G: 255, B: 255, A: 255})
}

func (storage VertexDataStorage) setCellPos(index int, pos F32Rect) {
	assert_debug(index >= 0 && storage.begin+index < storage.end)
	storage.renderer.vertexData[storage.begin+index].pos = pos
}

func (storage VertexDataStorage) setCellTex(index int, texPos IntRect) {
	assert_debug(index >= 0 && storage.begin+index < storage.end)
	tex1pos := storage.renderer.fontAtlas.texture.glCoords(texPos)
	for i := 0; i < 4; i++ {
		storage.renderer.vertexData[storage.begin+index].tex1 = tex1pos
	}
}

func (storage VertexDataStorage) setCellTex2(index int, texPos IntRect) {
	assert_debug(index >= 0 && storage.begin+index < storage.end)
	tex2pos := storage.renderer.fontAtlas.texture.glCoords(texPos)
	storage.renderer.vertexData[storage.begin+index].tex2 = tex2pos
}

func (storage VertexDataStorage) setCellFg(index int, fg U8Color) {
	assert_debug(index >= 0 && storage.begin+index < storage.end)
	storage.renderer.vertexData[storage.begin+index].fg = fg.toF32()
}

func (storage VertexDataStorage) setCellBg(index int, bg U8Color) {
	assert_debug(index >= 0 && storage.begin+index < storage.end)
	storage.renderer.vertexData[storage.begin+index].bg = bg.toF32()
}

func (storage VertexDataStorage) setCellSp(index int, sp U8Color) {
	assert_debug(index >= 0 && storage.begin+index < storage.end)
	storage.renderer.vertexData[storage.begin+index].sp = sp.toF32()
}

// Reserve calculates needed vertex size for given cell count,
// allocates data for it and returns beginning of the index of the reserved data.
// You can set this data using SetVertex* functions. Functions takes index arguments
// as cell positions, not vertex data positions.
func (renderer *Renderer) reserveVertexData(cellCount int) VertexDataStorage {
	begin := len(renderer.vertexData)
	for i := 0; i < cellCount; i++ {
		renderer.vertexData = append(renderer.vertexData, Vertex{})
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
		dst_data.tex2 = src_data.tex2
		dst_data.tex1 = src_data.tex1
		dst_data.fg = src_data.fg
		dst_data.bg = src_data.bg
		dst_data.sp = src_data.sp
	}
}

func (renderer *Renderer) setCellTex(x, y int, pos IntRect) {
	tex1pos := renderer.fontAtlas.texture.glCoords(pos)
	renderer.vertexData[cellVertexPos(x, y)].tex1 = tex1pos
}

func (renderer *Renderer) setCellTex2(x, y int, pos IntRect) {
	tex2pos := renderer.fontAtlas.texture.glCoords(pos)
	renderer.vertexData[cellVertexPos(x, y)].tex2 = tex2pos
}

func (renderer *Renderer) setCellFg(x, y int, fg U8Color) {
	renderer.vertexData[cellVertexPos(x, y)].fg = fg.toF32()
}

func (renderer *Renderer) setCellBg(x, y int, bg U8Color) {
	renderer.vertexData[cellVertexPos(x, y)].bg = bg.toF32()
}

func (renderer *Renderer) setCellSp(x, y int, sp U8Color) {
	renderer.vertexData[cellVertexPos(x, y)].sp = sp.toF32()
}

func (renderer *Renderer) debugGetCellData(x, y int) Vertex {
	return renderer.vertexData[cellVertexPos(x, y)]
}

func cellPos(x, y int) F32Rect {
	return F32Rect{
		X: float32(y * EditorSingleton.cellWidth),
		Y: float32(x * EditorSingleton.cellHeight),
		W: float32(EditorSingleton.cellWidth),
		H: float32(EditorSingleton.cellHeight),
	}
}

func cellVertexPos(x, y int) int {
	return x*EditorSingleton.columnCount + y
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
		// The clear atlas function calls a popup menu function to create its
		// chars. This means the nextAtlasPosition is recursively calling
		// itself when the font atlas is full. And the atlas pos is changed.
		// We need to reset pos to atlas pos for resuming from where it is.
		// Even if the popup menu function is not called, the clear atlas
		// function will clear the atlas pos and ve resume from beginning.
		// NOTE: When the current characters on the screen can't fit the texture
		// the characters being wrongly positioned. It doesn't crash but sometimes
		// the position is going out of bounds from texture. And assertion will fail.
		// This only happens in debug mode to me because texture size is small in debug.
		pos = atlas.pos
	}
	atlas.pos.X = pos.X + width
	atlas.pos.Y = pos.Y
	assert(pos.X+width < FONT_ATLAS_DEFAULT_SIZE, "Pos width out of bounds, pos:", pos, "width:", width)
	assert(pos.Y+EditorSingleton.cellHeight < FONT_ATLAS_DEFAULT_SIZE, "Pos height out of bounds, pos:", pos)
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

func (renderer *Renderer) update() {
	if renderer.drawCall {
		renderer.drawAllChangedCells()
		renderer.drawCall = false
	}
	if renderer.renderCall {
		renderer.render()
		renderer.renderCall = false
	}
}

// This function sets renderCall to true and draws cursor one more time.
// Dont use directly. Use EditorSingleton.draw()
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

// Don't call this function directly. Use EditorSingleton.render()
func (renderer *Renderer) render() {
	rglUpdateVertices(renderer.vertexData)
	rglClearScreen(EditorSingleton.grid.defaultBg)
	rglRender()
	EditorSingleton.window.handle.SwapBuffers()
}

func (renderer *Renderer) Close() {
	renderer.fontAtlas.texture.Delete()
	rglClose()
}
