package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

const (
	CursorAnimationLifetime = 0.4
)

type Cursor struct {
	X            int
	Y            int
	anim         Animation
	needs_redraw bool
	hidden       bool
}

func (cursor *Cursor) Update() {
	if cursor.needs_redraw {
		cursor.Draw()
	}
}

func (cursor *Cursor) SetPosition(x, y int, immediately bool) {
	if !immediately {
		cursor.anim = CreateAnimation(
			f32vec2{X: float32(cursor.X), Y: float32(cursor.Y)},
			f32vec2{X: float32(x), Y: float32(y)},
			CursorAnimationLifetime)
	}
	cursor.X = x
	cursor.Y = y
	cursor.needs_redraw = true
}

func (cursor *Cursor) GetPositionRectangle(cell_pos ivec2, info ModeInfo) (sdl.Rect, bool) {
	var cursor_rect sdl.Rect
	var draw_char bool
	switch info.cursor_shape {
	case "block":
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y),
			W: int32(EditorSingleton.cellWidth),
			H: int32(EditorSingleton.cellHeight),
		}
		draw_char = true
		break
	case "horizontal":
		height := EditorSingleton.cellHeight / (100 / info.cell_percentage)
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y + (EditorSingleton.cellHeight - height)),
			W: int32(EditorSingleton.cellWidth),
			H: int32(height),
		}
		break
	case "vertical":
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y),
			W: int32(EditorSingleton.cellWidth / (100 / info.cell_percentage)),
			H: int32(EditorSingleton.cellHeight),
		}
		break
	}
	return cursor_rect, draw_char
}

func (cursor *Cursor) GetColors(info ModeInfo) (sdl.Color, sdl.Color) {
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

func (cursor *Cursor) GetAnimatedPosition() ivec2 {
	aPos, finished := cursor.anim.GetCurrentStep(EditorSingleton.deltaTime)
	if finished {
		cursor.needs_redraw = false
		return ivec2{
			X: EditorSingleton.cellWidth * cursor.Y,
			Y: EditorSingleton.cellHeight * cursor.X,
		}
	} else {
		return ivec2{
			X: int(float32(EditorSingleton.cellWidth) * aPos.Y),
			Y: int(float32(EditorSingleton.cellHeight) * aPos.X),
		}
	}
}

func (cursor *Cursor) Draw() {
	defer measure_execution_time("Cursor.Draw")()
	if cursor.hidden {
		// Hide the cursor.
		EditorSingleton.renderer.SetCursorData(
			sdl.Rect{}, sdl.Rect{}, sdl.Color{}, sdl.Color{})
	} else {
		mode_info := EditorSingleton.mode.mode_infos[EditorSingleton.mode.current_mode_name]
		fg, bg := cursor.GetColors(mode_info)
		pos := cursor.GetAnimatedPosition()
		rect, draw_char := cursor.GetPositionRectangle(pos, mode_info)
		if draw_char && !cursor.needs_redraw {
			cell := EditorSingleton.grid.cells[cursor.X][cursor.Y]
			if len(cell.char) != 0 && cell.char != " " {
				// We need to draw cell character to the cursor foreground.
				// Because cursor is not transparent.
				italic := false
				bold := false
				if cell.attrib_id > 0 {
					attrib := EditorSingleton.grid.attributes[cell.attrib_id]
					italic = attrib.italic
					bold = attrib.bold
				}
				atlas_pos, err := EditorSingleton.renderer.GetCharacterAtlasPosition(cell.char, italic, bold)
				if err != nil {
					return
				}
				EditorSingleton.renderer.SetCursorData(rect, atlas_pos, fg, bg)
			}
		} else {
			// No cell drawing needed. Just draw the cursor.
			EditorSingleton.renderer.SetCursorData(
				rect, sdl.Rect{}, sdl.Color{}, bg)
		}
	}
	EditorSingleton.renderer.Render()
}
