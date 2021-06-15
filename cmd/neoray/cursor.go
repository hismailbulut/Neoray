package main

import ()

const (
	CursorAnimationLifetime = 0.3
)

type Cursor struct {
	X         int
	Y         int
	anim      Animation
	needsDraw bool
	hidden    bool
}

func (cursor *Cursor) Update() {
	if cursor.needsDraw {
		cursor.Draw()
	}
}

func (cursor *Cursor) IsHere(x, y int) bool {
	return x == cursor.X && y == cursor.Y
}

func (cursor *Cursor) SetPosition(x, y int, immediately bool) {
	if !immediately {
		cursor.anim = CreateAnimation(
			F32Vec2{X: float32(cursor.X), Y: float32(cursor.Y)},
			F32Vec2{X: float32(x), Y: float32(y)},
			CursorAnimationLifetime)
	}
	cursor.X = x
	cursor.Y = y
	cursor.needsDraw = true
}

func (cursor *Cursor) GetRectangle(cell_pos IntVec2, info ModeInfo) (IntRect, bool) {
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

func (cursor *Cursor) GetColors(info ModeInfo) (U8Color, U8Color) {
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

func (cursor *Cursor) GetAnimatedPosition() IntVec2 {
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
	// Hide the cursor.
	EditorSingleton.renderer.SetCursorData(IntRect{}, IntRect{}, U8Color{}, U8Color{})
	cursor.hidden = true
}

func (cursor *Cursor) Draw() {
	defer measure_execution_time("Cursor.Draw")()
	if !cursor.hidden {
		mode_info := EditorSingleton.mode.mode_infos[EditorSingleton.mode.current_mode_name]
		fg, bg := cursor.GetColors(mode_info)
		pos := cursor.GetAnimatedPosition()
		rect, draw_char := cursor.GetRectangle(pos, mode_info)
		if draw_char && !cursor.needsDraw {
			cell := EditorSingleton.grid.GetCell(cursor.X, cursor.Y)
			if cell == nil {
				log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "No cell founded at cursor position.")
				return
			}
			if cell.char != "" && cell.char != " " {
				// We need to draw cell character to the cursor foreground.
				// Because cursor is not transparent.
				italic := false
				bold := false
				if cell.attrib_id > 0 {
					attrib := EditorSingleton.grid.attributes[cell.attrib_id]
					italic = attrib.italic
					bold = attrib.bold
				}
				atlas_pos := EditorSingleton.renderer.GetCharacterAtlasPosition(cell.char, italic, bold)
				EditorSingleton.renderer.SetCursorData(rect, atlas_pos, fg, bg)
			}
		} else {
			// No cell drawing needed. Just draw the cursor.
			EditorSingleton.renderer.SetCursorData(
				rect, IntRect{}, U8Color{}, bg)
		}
		EditorSingleton.renderer.renderCall = true
	}
}
