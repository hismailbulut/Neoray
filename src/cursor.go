package main

type Cursor struct {
	X          int
	Y          int
	anim       Animation
	needsDraw  bool
	hidden     bool
	vertexData VertexDataStorage
	// blinking variables
	bHidden  bool
	time     float64
	nextTime float64
}

func CreateCursor() Cursor {
	return Cursor{}
}

func (cursor *Cursor) update() {
	cursor.time += EditorSingleton.deltaTime
	// Blinking
	cursor.updateBlinking()
	// Draw cursor if it needs.
	if cursor.needsDraw {
		cursor.Draw()
	}
}

func (cursor *Cursor) resetBlinking() {
	info := EditorSingleton.mode.Current()
	// When one of the numbers is zero, there is no blinking.
	if info.blinkwait <= 0 || info.blinkon <= 0 || info.blinkoff <= 0 {
		return
	}
	cursor.time = 0
	cursor.blinkShow()
	cursor.nextTime = float64(info.blinkwait) / 1000
}

func (cursor *Cursor) updateBlinking() {
	info := EditorSingleton.mode.Current()
	// When one of the numbers is zero, there is no blinking.
	if info.blinkwait <= 0 || info.blinkon <= 0 || info.blinkoff <= 0 {
		return
	}
	if cursor.time >= cursor.nextTime {
		cursor.time = 0
		if cursor.bHidden {
			// show cursor
			cursor.blinkShow()
			cursor.nextTime = float64(info.blinkon) / 1000
		} else {
			// hide cursor
			cursor.blinkHide()
			cursor.nextTime = float64(info.blinkoff) / 1000
		}
	}
}

func (cursor *Cursor) createVertexData() {
	// Reserve vertex data for cursor and cursor is only one cell.
	cursor.vertexData = EditorSingleton.renderer.reserveVertexData(1)
}

func (cursor *Cursor) SetPosition(x, y int, immediately bool) {
	assert(x >= 0 && y >= 0 &&
		x < EditorSingleton.rowCount && y < EditorSingleton.columnCount,
		"cursor pos incorrect", x, y, immediately)
	if !immediately {
		cursor.anim = CreateAnimation(
			F32Vec2{X: float32(cursor.X), Y: float32(cursor.Y)},
			F32Vec2{X: float32(x), Y: float32(y)},
			EditorSingleton.options.cursorAnimTime)
	}
	cursor.X = x
	cursor.Y = y
	cursor.needsDraw = true
	cursor.resetBlinking()
}

func (cursor *Cursor) isInArea(x, y, w, h int) bool {
	return cursor.X >= x && cursor.Y >= y &&
		cursor.X < x+w && cursor.Y < y+h
}

func (cursor *Cursor) modeRectangle(cell_pos IntVec2, info ModeInfo) (F32Rect, bool) {
	switch info.cursor_shape {
	case "block":
		return F32Rect{
			X: float32(cell_pos.X),
			Y: float32(cell_pos.Y),
			W: float32(EditorSingleton.cellWidth),
			H: float32(EditorSingleton.cellHeight),
		}, true
	case "horizontal":
		height := float32(EditorSingleton.cellHeight) / (100 / float32(info.cell_percentage))
		return F32Rect{
			X: float32(cell_pos.X),
			Y: float32(cell_pos.Y) + (float32(EditorSingleton.cellHeight) - height),
			W: float32(EditorSingleton.cellWidth),
			H: height,
		}, false
	case "vertical":
		return F32Rect{
			X: float32(cell_pos.X),
			Y: float32(cell_pos.Y),
			W: float32(EditorSingleton.cellWidth) / (100 / float32(info.cell_percentage)),
			H: float32(EditorSingleton.cellHeight),
		}, false
	default:
		return F32Rect{}, false
	}
}

func (cursor *Cursor) modeColors(info ModeInfo) (U8Color, U8Color) {
	// initialize swapped
	fg := EditorSingleton.grid.defaultBg
	bg := EditorSingleton.grid.defaultFg
	if info.attr_id != 0 {
		attrib := EditorSingleton.grid.attributes[info.attr_id]
		if attrib.foreground.A > 0 {
			fg = attrib.foreground
		}
		if attrib.background.A > 0 {
			bg = attrib.background
		}
	}
	return fg, bg
}

// animPosition returns the current rendering position of the cursor
// Not grid position.
func (cursor *Cursor) animPosition() IntVec2 {
	aPos, finished := cursor.anim.GetCurrentStep(float32(EditorSingleton.deltaTime))
	if finished {
		cursor.needsDraw = false
		return IntVec2{
			X: EditorSingleton.cellWidth * cursor.Y,
			Y: EditorSingleton.cellHeight * cursor.X,
		}
	} else {
		return IntVec2{
			X: int(float32(EditorSingleton.cellWidth) * aPos.Y),
			Y: int(float32(EditorSingleton.cellHeight) * aPos.X),
		}
	}
}

func (cursor *Cursor) blinkShow() {
	if !cursor.hidden && cursor.bHidden {
		cursor.bHidden = false
		cursor.Draw()
	}
}

func (cursor *Cursor) blinkHide() {
	if !cursor.hidden && !cursor.bHidden {
		cursor.bHidden = true
		cursor.vertexData.setCellPos(0, F32Rect{})
		EditorSingleton.render()
	}
}

func (cursor *Cursor) Show() {
	if cursor.hidden {
		cursor.hidden = false
		cursor.Draw()
	}
}

func (cursor *Cursor) Hide() {
	if !cursor.hidden {
		cursor.hidden = true
		cursor.vertexData.setCellPos(0, F32Rect{})
		EditorSingleton.render()
	}
}

func (cursor *Cursor) drawWithCell(cell Cell, fg U8Color) {
	italic := false
	bold := false
	underline := false
	strikethrough := false
	if cell.attribId > 0 {
		attrib := EditorSingleton.grid.attributes[cell.attribId]
		italic = attrib.italic
		bold = attrib.bold
		underline = attrib.underline
		strikethrough = attrib.strikethrough
		if attrib.undercurl {
			cursor.vertexData.setCellSp(0, fg)
		}
	}
	atlas_pos := EditorSingleton.renderer.getCharPos(
		cell.char, italic, bold, underline, strikethrough)
	if atlas_pos.W > EditorSingleton.cellWidth {
		atlas_pos.W /= 2
	}
	cursor.vertexData.setCellTex(0, atlas_pos)
}

func (cursor *Cursor) Draw() {
	if !cursor.hidden {
		mode_info := EditorSingleton.mode.Current()
		fg, bg := cursor.modeColors(mode_info)
		pos := cursor.animPosition()
		rect, draw_char := cursor.modeRectangle(pos, mode_info)
		if draw_char && !cursor.needsDraw {
			cell := EditorSingleton.grid.getCell(cursor.X, cursor.Y)
			if cell.char != 0 {
				// We need to draw cell character to the cursor foreground.
				cursor.drawWithCell(cell, fg)
			} else {
				// Clear foreground of the cursor.
				cursor.vertexData.setCellTex(0, IntRect{})
				cursor.vertexData.setCellSp(0, U8Color{})
			}
			cursor.vertexData.setCellFg(0, fg)
			cursor.vertexData.setCellBg(0, bg)
		} else {
			// No cell drawing needed. Clear foreground.
			cursor.vertexData.setCellTex(0, IntRect{})
			cursor.vertexData.setCellFg(0, U8Color{})
			cursor.vertexData.setCellBg(0, bg)
			cursor.vertexData.setCellSp(0, U8Color{})
		}
		cursor.vertexData.setCellPos(0, rect)
		EditorSingleton.render()
	}
}
