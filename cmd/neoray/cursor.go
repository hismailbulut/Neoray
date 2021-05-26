package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

type ModeInfo struct {
	cursor_shape    string
	cell_percentage int
	blinkwait       int
	blinkon         int
	blinkoff        int
	attr_id         int
	attr_id_lm      int
	short_name      string
	name            string
}

type Mode struct {
	cursor_style_enabled bool
	mode_infos           map[string]ModeInfo
	current_mode_name    string
	current_mode         int
}

type CursorDrawInfo struct {
	x, y      int
	fg, bg    sdl.Color
	rect      sdl.Rect
	draw_char bool
}

// TODO: Cursor animation.
type Cursor struct {
	X            int
	Y            int
	anim         Animation
	needs_redraw bool
}

func CreateMode() Mode {
	return Mode{
		mode_infos: make(map[string]ModeInfo),
	}
}

func (cursor *Cursor) Update(editor *Editor) {
	if cursor.needs_redraw {
		editor.renderer.DrawCursor(editor)
	}
}

func (cursor *Cursor) SetPosition(x, y int) {
	cursor.anim = CreateAnimation(
		f32vec2{X: float32(cursor.X), Y: float32(cursor.Y)},
		f32vec2{X: float32(x), Y: float32(y)},
		0.3)
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
			W: int32(GLOB_CellWidth),
			H: int32(GLOB_CellHeight),
		}
		draw_char = true
		break
	case "horizontal":
		height := GLOB_CellHeight / (100 / info.cell_percentage)
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y + (GLOB_CellHeight - height)),
			W: int32(GLOB_CellWidth),
			H: int32(height),
		}
		break
	case "vertical":
		cursor_rect = sdl.Rect{
			X: int32(cell_pos.X),
			Y: int32(cell_pos.Y),
			W: int32(GLOB_CellWidth / (100 / info.cell_percentage)),
			H: int32(GLOB_CellHeight),
		}
		break
	}
	return cursor_rect, draw_char
}

func (cursor *Cursor) GetColors(info ModeInfo, grid *Grid) (sdl.Color, sdl.Color) {
	// initialize swapped
	fg := grid.default_bg
	bg := grid.default_fg
	if info.attr_id != 0 {
		attrib := grid.attributes[info.attr_id]
		fg = attrib.foreground
		bg = attrib.background
	}
	return fg, bg
}

func (cursor *Cursor) GetAnimatedPosition() ivec2 {
	aPos := cursor.anim.GetCurrentStep()
	pos := ivec2{
		X: int(float32(GLOB_CellWidth) * aPos.Y),
		Y: int(float32(GLOB_CellHeight) * aPos.X),
	}
	cursor.needs_redraw = !(pos.X == cursor.X && pos.Y == cursor.Y)
	return pos
}

func (cursor *Cursor) GetDrawInfo(mode *Mode, grid *Grid) CursorDrawInfo {
	mode_info := mode.mode_infos[mode.current_mode_name]

	fg, bg := cursor.GetColors(mode_info, grid)

	cell_pos := cursor.GetAnimatedPosition()

	cursor_rect, draw_char :=
		cursor.GetPositionRectangle(cell_pos, mode_info)

	return CursorDrawInfo{
		x:         cursor.X,
		y:         cursor.Y,
		fg:        fg,
		bg:        bg,
		rect:      cursor_rect,
		draw_char: draw_char,
	}
}
