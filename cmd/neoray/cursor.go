package main

import ()

const (
	DefaultCursorAnimLifetime = 0.08
)

type Cursor struct {
	X            int
	Y            int
	anim         Animation
	animLifetime float32
	needsDraw    bool
	hidden       bool
	vertexData   VertexDataStorage
}

func CreateCursor() Cursor {
	return Cursor{
		animLifetime: DefaultCursorAnimLifetime,
	}
}

func (cursor *Cursor) Update() {
	if cursor.needsDraw {
		cursor.Draw()
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
			F32Vec2{X: float32(x), Y: float32(y)}, cursor.animLifetime)
	}
	cursor.X = x
	cursor.Y = y
	cursor.needsDraw = true
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
		height := EditorSingleton.cellHeight / (100 / info.cell_percentage)
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
			W: EditorSingleton.cellWidth / (100 / info.cell_percentage),
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
		fg = attrib.foreground
		bg = attrib.background
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
	cursor.hidden = false
	cursor.Draw()
}

func (cursor *Cursor) Hide() {
	cursor.hidden = true
	cursor.vertexData.setCellPos(0, IntRect{})
}

func (cursor *Cursor) drawWithCell(cell Cell, fg U8Color) {
	italic := false
	bold := false
	underline := false
	strikethrough := false
	if cell.attrib_id > 0 {
		attrib := EditorSingleton.grid.attributes[cell.attrib_id]
		italic = attrib.italic
		bold = attrib.bold
		underline = attrib.underline
		strikethrough = attrib.strikethrough
		if attrib.undercurl {
			cursor.vertexData.setCellSpColor(0, fg)
		}
	}
	atlas_pos := EditorSingleton.renderer.getCharPos(
		cell.char, italic, bold, underline, strikethrough)
	cursor.vertexData.setCellTexPos(0, atlas_pos)
}

func (cursor *Cursor) Draw() {
	defer measure_execution_time("Cursor.Draw")()
	if !cursor.hidden {
		mode_info := EditorSingleton.mode.CurrentModeInfo()
		fg, bg := cursor.modeColors(mode_info)
		pos := cursor.animPosition()
		rect, draw_char := cursor.modeRectangle(pos, mode_info)
		if draw_char && !cursor.needsDraw {
			cell := EditorSingleton.grid.GetCell(cursor.X, cursor.Y)
			if cell.char != 0 {
				// We need to draw cell character to the cursor foreground.
				cursor.drawWithCell(cell, fg)
			} else {
				// Clear foreground of the cursor.
				cursor.vertexData.setCellTexPos(0, IntRect{})
				cursor.vertexData.setCellSpColor(0, U8Color{})
			}
			cursor.vertexData.setCellColor(0, fg, bg)
		} else {
			// No cell drawing needed. Clear foreground.
			cursor.vertexData.setCellTexPos(0, IntRect{})
			cursor.vertexData.setCellColor(0, U8Color{}, bg)
			cursor.vertexData.setCellSpColor(0, U8Color{})
		}
		cursor.vertexData.setCellPos(0, rect)
		EditorSingleton.render()
	}
}
