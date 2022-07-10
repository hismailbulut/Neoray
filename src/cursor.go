package main

import (
	"github.com/hismailbulut/neoray/src/bench"
	"github.com/hismailbulut/neoray/src/common"
	"github.com/hismailbulut/neoray/src/logger"
	"github.com/hismailbulut/neoray/src/opengl"
	"github.com/hismailbulut/neoray/src/window"
)

type Cursor struct {
	row, col int              // Position of the cursor in the grid
	grid     int              // Id of the grid where the cursor is
	mode     Mode             // Current mode and style information (normal, visual etc.)
	anim     common.Animation // Cursor animation
	hidden   bool
	// TODO: We can make a cursor renderer with different features
	buffer *opengl.VertexBuffer
	// blinking variables
	bHidden  bool
	time     float32
	nextTime float32
}

func NewCursor(window *window.Window) *Cursor {
	cursor := new(Cursor)
	cursor.buffer = window.GL().CreateVertexBuffer(1)
	return cursor
}

func (cursor *Cursor) Update(delta float32) {
	cursor.time += delta
	if cursor.anim.IsFinished() {
		// Do not blink while animating
		cursor.updateBlinking()
	} else {
		// Additional draw call to cursor for animation
		// TODO We don't need to draw whole screen, just cursor enough
		MarkDraw()
	}
}

func (cursor *Cursor) Show() {
	if cursor.hidden {
		cursor.hidden = false
		MarkRender()
	}
}

func (cursor *Cursor) Hide() {
	if !cursor.hidden {
		cursor.hidden = true
		MarkRender()
	}
}

// Returns the grid where the cursor is
func (cursor *Cursor) Grid() *Grid {
	return Editor.gridManager.Grid(cursor.grid)
}

func (cursor *Cursor) resetBlinking() {
	info := cursor.mode.Current()
	// When one of the numbers is zero, there is no blinking.
	if info.blinkwait <= 0 || info.blinkon <= 0 || info.blinkoff <= 0 {
		return
	}
	cursor.time = 0
	cursor.nextTime = float32(info.blinkwait) / 1000
	cursor.blinkShow()
}

func (cursor *Cursor) updateBlinking() {
	info := cursor.mode.Current()
	// When one of the numbers is zero, there is no blinking.
	if info.blinkwait <= 0 || info.blinkon <= 0 || info.blinkoff <= 0 {
		return
	}
	if cursor.time >= cursor.nextTime {
		cursor.time = 0
		if cursor.bHidden {
			// show cursor
			cursor.blinkShow()
			cursor.nextTime = float32(info.blinkon) / 1000
		} else {
			// hide cursor
			cursor.blinkHide()
			cursor.nextTime = float32(info.blinkoff) / 1000
		}
	}
}

func (cursor *Cursor) blinkShow() {
	if cursor.bHidden {
		cursor.bHidden = false
		MarkRender()
	}
}

func (cursor *Cursor) blinkHide() {
	if !cursor.bHidden {
		cursor.bHidden = true
		MarkRender()
	}
}

func (cursor *Cursor) SetPosition(id, row, col int, immediately bool) {
	if !immediately {
		func() {
			// find global row and column of the cursor for both current and target
			currentGrid := cursor.Grid()
			if currentGrid == nil {
				return
			}
			current := common.Vector2[float32]{
				X: float32(currentGrid.PixelPos().X + (cursor.col * currentGrid.CellSize().Width())),
				Y: float32(currentGrid.PixelPos().Y + (cursor.row * currentGrid.CellSize().Height())),
			}
			targetGrid := Editor.gridManager.Grid(id)
			if targetGrid == nil {
				return
			}
			target := common.Vector2[float32]{
				X: float32(targetGrid.PixelPos().X + (col * targetGrid.CellSize().Width())),
				Y: float32(targetGrid.PixelPos().Y + (row * targetGrid.CellSize().Height())),
			}
			cursor.anim = common.NewAnimation(current, target, Editor.options.cursorAnimTime)
		}()
	}
	cursor.grid = id
	cursor.row = row
	cursor.col = col
	cursor.resetBlinking()
	MarkDraw()
}

func (cursor *Cursor) IsInArea(grid, x, y, w, h int) bool {
	return grid == cursor.grid && cursor.row >= x && cursor.col >= y && cursor.row < x+w && cursor.col < y+h
}

func (cursor *Cursor) modeRectangle(info ModeInfo, position, cellSize common.Vector2[int]) (common.Rectangle[float32], bool) {
	switch info.cursor_shape {
	case "block":
		return common.Rectangle[float32]{
			X: float32(position.X),
			Y: float32(position.Y),
			W: float32(cellSize.Width()),
			H: float32(cellSize.Height()),
		}, true
	case "horizontal":
		height := float32(cellSize.Height()) / (100 / float32(info.cell_percentage))
		return common.Rectangle[float32]{
			X: float32(position.X),
			Y: float32(position.Y) + (float32(cellSize.Height()) - height),
			W: float32(cellSize.Width()),
			H: height,
		}, false
	case "vertical":
		return common.Rectangle[float32]{
			X: float32(position.X),
			Y: float32(position.Y),
			W: float32(cellSize.Width()) / (100 / float32(info.cell_percentage)),
			H: float32(cellSize.Height()),
		}, false
	default:
		return common.ZeroRectangleF32, false
	}
}

func (cursor *Cursor) AttributeColors(id int) (common.Color[uint8], common.Color[uint8]) {
	// When attr_id is 0 we are using cursor foreground for default background and background for default foreground
	fg := Editor.gridManager.background
	bg := Editor.gridManager.foreground
	if id != 0 {
		attrib, ok := Editor.gridManager.attributes[id]
		if ok {
			if attrib.foreground.A > 0 {
				fg = attrib.foreground
			}
			if attrib.background.A > 0 {
				bg = attrib.background
			}
		}
	}
	return fg, bg
}

func (cursor *Cursor) Draw(delta float32) {
	if cursor.hidden || cursor.bHidden {
		return
	}
	EndBenchmark := bench.BeginBenchmark()
	defer EndBenchmark("Cursor.Draw")
	// Draw
	modeInfo := cursor.mode.Current()
	cursorFg, cursorBg := cursor.AttributeColors(modeInfo.attr_id)
	// Current grid where the cursor is
	grid := cursor.Grid()
	if grid != nil {
		pos := cursor.anim.Step(delta).ToInt()
		rect, blockShaped := cursor.modeRectangle(modeInfo, pos, grid.CellSize())
		cell := grid.CellAt(cursor.row, cursor.col)
		// Only draw character to the cursor if animation is finished and cell
		// has a printable character and cursor shape is block
		if cursor.anim.IsFinished() && cell.char != 0 && blockShaped {
			// We need to draw cell character to the cursor foreground
			cellAttrib := Editor.gridManager.Attribute(cell.attribID)
			// Draw undercurl to cursor if cell has
			if cellAttrib.undercurl {
				cursor.buffer.SetIndexSp(0, cursorFg.ToF32())
			}
			// Draw this cell to the cursor
			charPos := grid.renderer.atlas.GetCharPos(cell.char, cellAttrib.bold, cellAttrib.italic, cellAttrib.underline, cellAttrib.strikethrough, grid.CellSize())
			if charPos.W > grid.CellSize().Width() {
				charPos.W /= 2
			}
			cursor.buffer.SetIndexTex1(0, grid.renderer.atlas.Normalize(charPos))
			cursor.buffer.SetIndexFg(0, cursorFg.ToF32())
		} else {
			// No cell drawing needed. Clear foreground.
			cursor.buffer.SetIndexTex1(0, common.ZeroRectangleF32)
			cursor.buffer.SetIndexFg(0, common.ZeroColorF32)
			cursor.buffer.SetIndexSp(0, common.ZeroColorF32)
		}
		// Background and position is always required
		cursor.buffer.SetIndexBg(0, cursorBg.ToF32())
		cursor.buffer.SetIndexPos(0, rect)
	}
}

func (cursor *Cursor) Render() {
	if cursor.hidden || cursor.bHidden {
		return
	}
	grid := cursor.Grid()
	if grid != nil {
		// Because we are drawing grid's characters, we need it's atlas
		grid.renderer.atlas.BindTexture()
		cursor.buffer.Bind()
		cursor.buffer.Update()
		// TODO Do we need to update projection?
		cursor.buffer.Render()
	}
}

func (cursor *Cursor) Destroy() {
	cursor.buffer.Destroy()
	logger.Log(logger.DEBUG, "Cursor destroyed")
}
