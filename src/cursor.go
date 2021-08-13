package main

type Cursor struct {
	X, Y       int
	grid       int
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
	cursor.time += singleton.deltaTime
	// Blinking
	cursor.updateBlinking()
	// Draw cursor if it needs.
	if cursor.needsDraw {
		cursor.Draw()
	}
}

func (cursor *Cursor) resetBlinking() {
	info := singleton.mode.Current()
	// When one of the numbers is zero, there is no blinking.
	if info.blinkwait <= 0 || info.blinkon <= 0 || info.blinkoff <= 0 {
		return
	}
	cursor.time = 0
	cursor.blinkShow()
	cursor.nextTime = float64(info.blinkwait) / 1000
}

func (cursor *Cursor) updateBlinking() {
	info := singleton.mode.Current()
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
	cursor.vertexData = singleton.renderer.reserveVertexData(1)
}

func (cursor *Cursor) setPosition(id, x, y int, immediately bool) {
	if !immediately {
		cursor.anim = CreateAnimation(
			F32Vec2{X: float32(cursor.X), Y: float32(cursor.Y)},
			F32Vec2{X: float32(x), Y: float32(y)},
			singleton.options.cursorAnimTime)
	}
	cursor.X = x
	cursor.Y = y
	cursor.grid = id
	cursor.needsDraw = true
	cursor.resetBlinking()
}

func (cursor *Cursor) isInArea(gridId, x, y, w, h int) bool {
	return gridId == cursor.grid && cursor.X >= x && cursor.Y >= y && cursor.X < x+w && cursor.Y < y+h
}

func (cursor *Cursor) modeRectangle(cell_pos IntVec2, info ModeInfo) (F32Rect, bool) {
	switch info.cursor_shape {
	case "block":
		return F32Rect{
			X: float32(cell_pos.X),
			Y: float32(cell_pos.Y),
			W: float32(singleton.cellWidth),
			H: float32(singleton.cellHeight),
		}, true
	case "horizontal":
		height := float32(singleton.cellHeight) / (100 / float32(info.cell_percentage))
		return F32Rect{
			X: float32(cell_pos.X),
			Y: float32(cell_pos.Y) + (float32(singleton.cellHeight) - height),
			W: float32(singleton.cellWidth),
			H: height,
		}, false
	case "vertical":
		return F32Rect{
			X: float32(cell_pos.X),
			Y: float32(cell_pos.Y),
			W: float32(singleton.cellWidth) / (100 / float32(info.cell_percentage)),
			H: float32(singleton.cellHeight),
		}, false
	default:
		return F32Rect{}, false
	}
}

func (cursor *Cursor) modeColors(info ModeInfo) (U8Color, U8Color) {
	// initialize swapped
	fg := singleton.gridManager.defaultBg
	bg := singleton.gridManager.defaultFg
	if info.attr_id != 0 {
		attrib := singleton.gridManager.attributes[info.attr_id]
		if attrib.foreground.A > 0 {
			fg = attrib.foreground
		}
		if attrib.background.A > 0 {
			bg = attrib.background
		}
	}
	return fg, bg
}

// This function returns the current rendering position of the cursor Not grid
// position. sRow and sCol are grid positions for adding to cursor position.
// Sets cursor.needsDraw to false when an animation finished.
func (cursor *Cursor) animPosition(sRow, sCol int) IntVec2 {
	aPos, finished := cursor.anim.GetCurrentStep(float32(singleton.deltaTime))
	if finished {
		cursor.needsDraw = false
		return IntVec2{
			X: singleton.cellWidth * (sCol + cursor.Y),
			Y: singleton.cellHeight * (sRow + cursor.X),
		}
	} else {
		return IntVec2{
			X: int(float32(singleton.cellWidth) * (float32(sCol) + aPos.Y)),
			Y: int(float32(singleton.cellHeight) * (float32(sRow) + aPos.X)),
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
		singleton.render()
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
		singleton.render()
	}
}

func (cursor *Cursor) drawWithCell(cell Cell, fg U8Color) {
	italic := false
	bold := false
	underline := false
	strikethrough := false
	if cell.attribId > 0 {
		attrib := singleton.gridManager.attributes[cell.attribId]
		italic = attrib.italic
		bold = attrib.bold
		underline = attrib.underline
		strikethrough = attrib.strikethrough
		if attrib.undercurl {
			cursor.vertexData.setCellSp(0, fg)
		}
	}
	atlas_pos := singleton.renderer.getCharPos(
		cell.char, italic, bold, underline, strikethrough)
	if atlas_pos.W > singleton.cellWidth {
		atlas_pos.W /= 2
	}
	cursor.vertexData.setCellTex1(0, atlas_pos)
}

func (cursor *Cursor) Draw() {
	if !cursor.hidden {
		mode_info := singleton.mode.Current()
		fg, bg := cursor.modeColors(mode_info)
		// Get global position of the cursor.
		sRow := 0
		sCol := 0
		grid, ok := singleton.gridManager.grids[cursor.grid]
		if ok {
			sRow = grid.sRow
			sCol = grid.sCol
		}
		pos := cursor.animPosition(sRow, sCol)
		rect, draw_char := cursor.modeRectangle(pos, mode_info)
		// if the draw_char is true, then the cursor shape is block
		// if the cursor.needsDraw is false, then the cursor animation is finished and this is the last draw
		if draw_char && !cursor.needsDraw && ok {
			cell := grid.getCell(cursor.X, cursor.Y)
			if cell.char != 0 {
				// We need to draw cell character to the cursor foreground.
				cursor.drawWithCell(cell, fg)
			} else {
				// Clear foreground character of the cursor.
				cursor.vertexData.setCellTex1(0, IntRect{})
				cursor.vertexData.setCellSp(0, U8Color{})
			}
			cursor.vertexData.setCellFg(0, fg)
			cursor.vertexData.setCellBg(0, bg)
		} else {
			// No cell drawing needed. Clear foreground.
			cursor.vertexData.setCellTex1(0, IntRect{})
			cursor.vertexData.setCellFg(0, U8Color{})
			cursor.vertexData.setCellBg(0, bg)
			cursor.vertexData.setCellSp(0, U8Color{})
		}
		cursor.vertexData.setCellPos(0, rect)
		singleton.render()
	}
}
