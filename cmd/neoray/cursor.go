package main

import ()

// DefaultCursorAnimLifetime = 0.08

type Cursor struct {
	X          int
	Y          int
	anim       Animation
	needsDraw  bool
	hidden     bool
	vertexData VertexDataStorage
	// blinking variables
	time     float32
	nextTime float32
}

func CreateCursor() Cursor {
	return Cursor{}
}

func (cursor *Cursor) Update() {
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
	cursor.Show()
	cursor.nextTime = float32(info.blinkwait) / 1000
}

func (cursor *Cursor) updateBlinking() {
	info := EditorSingleton.mode.Current()
	// When one of the numbers is zero, there is no blinking.
	if info.blinkwait <= 0 || info.blinkon <= 0 || info.blinkoff <= 0 {
		return
	}
	if cursor.time >= cursor.nextTime {
		cursor.time = 0
		if cursor.hidden {
			// show cursor
			cursor.Show()
			cursor.nextTime = float32(info.blinkon) / 1000
		} else {
			// hide cursor
			cursor.Hide()
			cursor.nextTime = float32(info.blinkoff) / 1000
		}
	}
}

func (cursor *Cursor) createVertexData() {
	// Reserve vertex data for cursor and cursor is only one cell.
	cursor.vertexData = EditorSingleton.renderer.reserveVertexData(1)
}

func (cursor *Cursor) SetPosition(x, y int, immediately bool) {
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

func (cursor *Cursor) modeRectangle(cell_pos IntVec2, info ModeInfo) (IntRect, bool) {
	var cursor_rect IntRect
	var draw_char bool
	switch info.cursor_shape {
	case "block":
		cursor_rect = IntRect{
			X: cell_pos.X,
			Y: cell_pos.Y,
			W: EditorSingleton.cellWidth,
			H: EditorSingleton.cellHeight,
		}
		draw_char = true
		break
	case "horizontal":
		height := int(float32(EditorSingleton.cellHeight) / (100 / float32(info.cell_percentage)))
		cursor_rect = IntRect{
			X: cell_pos.X,
			Y: cell_pos.Y + (EditorSingleton.cellHeight - height),
			W: EditorSingleton.cellWidth,
			H: height,
		}
		break
	case "vertical":
		cursor_rect = IntRect{
			X: cell_pos.X,
			Y: cell_pos.Y,
			W: int(float32(EditorSingleton.cellWidth) / (100 / float32(info.cell_percentage))),
			H: EditorSingleton.cellHeight,
		}
		break
	}
	return cursor_rect, draw_char
}

func (cursor *Cursor) modeColors(info ModeInfo) (U8Color, U8Color) {
	// initialize swapped
	fg := EditorSingleton.grid.default_bg
	bg := EditorSingleton.grid.default_fg
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
	aPos, finished := cursor.anim.GetCurrentStep(EditorSingleton.deltaTime)
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

func (cursor *Cursor) Show() {
	if cursor.hidden {
		cursor.hidden = false
		cursor.Draw()
	}
}

func (cursor *Cursor) Hide() {
	if !cursor.hidden {
		cursor.hidden = true
		cursor.vertexData.setCellPos(0, IntRect{})
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
	defer measure_execution_time()()
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
