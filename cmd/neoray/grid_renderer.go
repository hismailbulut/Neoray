package main

import (
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/fontkit"
	"github.com/hismailbulut/Neoray/pkg/opengl"
	"github.com/hismailbulut/Neoray/pkg/window"
)

type GridRenderer struct {
	atlas    *opengl.Atlas        // Font atlas of this renderer
	buffer   *opengl.VertexBuffer // Vertex buffer of this renderer
	position common.Vector2[int]
	rows     int
	cols     int
}

func NewGridRenderer(window *window.Window, rows, cols int, kit *fontkit.FontKit, fontSize float64, position common.Vector2[int]) (*GridRenderer, error) {
	renderer := new(GridRenderer)
	renderer.atlas = window.GL().NewAtlas(kit, fontSize, window.DPI(), Editor.options.boxDrawingEnabled, Editor.options.boxDrawingEnabled)
	renderer.buffer = window.GL().CreateVertexBuffer(rows * cols)
	renderer.rows = rows
	renderer.cols = cols
	renderer.position = position
	renderer.UpdatePositions()
	return renderer, nil
}

func (renderer *GridRenderer) SetFontKit(kit *fontkit.FontKit) {
	renderer.atlas.SetFontKit(kit)
	renderer.UpdatePositions()
}

func (renderer *GridRenderer) FontSize() float64 {
	return renderer.atlas.FontSize()
}

func (renderer *GridRenderer) SetFontSize(fontSize, dpi float64) {
	renderer.atlas.SetFontSize(fontSize, dpi)
	renderer.UpdatePositions()
}

func (renderer *GridRenderer) SetBoxDrawing(useBoxDrawing, useBlockDrawing bool) {
	renderer.atlas.SetBoxDrawing(useBoxDrawing, useBlockDrawing)
}

func (renderer *GridRenderer) SetPos(position common.Vector2[int]) {
	renderer.position = position
	renderer.UpdatePositions()
}

func (renderer *GridRenderer) Resize(rows, cols int) {
	renderer.rows = rows
	renderer.cols = cols
	renderer.buffer.Resize(rows * cols)
	renderer.UpdatePositions()
}

func (renderer *GridRenderer) CellSize() common.Vector2[int] {
	return renderer.atlas.ImageSize()
}

// NOTE: Neovim's coordinates and opengl coordinates we are using are
// different. The starting positions are same, top left corner. But neovim sends
// cell info as row, column based. First sends row and second column. But
// opengl uses row as y and column as x. First needs column and second needs
// row. We are storing data like neovim and because of this we need to multiply
// position with other axis.
// Neovim:
//     +-----> Column, y, second
//     |
//     v Row, x, first
// Opengl:
//     +-----> Column, x, first
//     |
//     v Row, y, second
// This function returns position rectangle of the cell needed for opengl.
func (renderer *GridRenderer) cellPos(row, col int, cellSize common.Vector2[int]) common.Rectangle[float32] {
	return common.Rectangle[float32]{
		X: float32(renderer.position.X + col*cellSize.Width()),
		Y: float32(renderer.position.Y + row*cellSize.Height()),
		W: float32(cellSize.Width()),
		H: float32(cellSize.Height()),
	}
}

func (renderer *GridRenderer) cellIndex(row, col int) int {
	return row*renderer.cols + col
}

func (renderer *GridRenderer) UpdatePositions() {
	cellSize := renderer.atlas.ImageSize()
	for row := 0; row < renderer.rows; row++ {
		for col := 0; col < renderer.cols; col++ {
			renderer.buffer.SetIndexPos(renderer.cellIndex(row, col), renderer.cellPos(row, col, cellSize))
		}
	}
}

func (renderer *GridRenderer) CellVertexData(row, col int) opengl.Vertex {
	return renderer.buffer.VertexAt(renderer.cellIndex(row, col))
}

func (renderer *GridRenderer) CopyRow(dst, src, left, right int) {
	dst_begin := renderer.cellIndex(dst, left)
	src_begin := renderer.cellIndex(src, left)
	src_end := renderer.cellIndex(src, right)
	for i := 0; i < src_end-src_begin; i++ {
		renderer.buffer.CopyButPos(dst_begin+i, src_begin+i)
	}
}

func (renderer *GridRenderer) DrawCell(row, col int, char rune, attrib HighlightAttribute) {
	// Calculate indices
	index := renderer.cellIndex(row, col)
	nextIndex := -1
	if col+1 < renderer.cols {
		nextIndex = renderer.cellIndex(row, col+1)
	}
	// Draw background
	renderer.buffer.SetIndexBg(index, attrib.background)

	if char == 0 {
		// This is an empty cell, clear foreground data (not color)
		// We will not clear foreground color because may the previous cell is a multiwidth character
		// and it may set the foreground color of this cell
		renderer.buffer.SetIndexTex1(index, common.ZeroRectangleF32)
		renderer.buffer.SetIndexSp(index, common.ZeroColor)
		if nextIndex != -1 {
			// Clear next cells second texture
			renderer.buffer.SetIndexTex2(nextIndex, common.ZeroRectangleF32)
		}
		return
	}

	cellSize := renderer.atlas.ImageSize()

	if attrib.undercurl {
		undercurlRect, firstDraw := renderer.atlas.Undercurl(cellSize)
		if firstDraw {
			// This is the first time we draw undercurl, because of this we must update
			// it's position to the shader
			// Buffer must bound while we updating undercurl rectangle
			renderer.buffer.Bind()
			renderer.buffer.SetUndercurlRect(renderer.atlas.Normalize(undercurlRect))
		}
		renderer.buffer.SetIndexSp(index, attrib.special)
	} else {
		// Setting special color to zero means clear the undercurl. Undercurl
		// will always be drawed for every cell and multiplied by the special
		// color. And setting special color to zero makes undercurl fully
		// transparent. This is also true for other color layouts.
		renderer.buffer.SetIndexSp(index, common.ZeroColor)
	}
	// Get character position in atlas texture
	atlasPos := renderer.atlas.GetCharPos(char, attrib.bold, attrib.italic, attrib.underline, attrib.strikethrough, cellSize)
	// Check if there is a require for second texture in next cell
	if atlasPos.W > cellSize.Width() {
		// The atlas width will be 2 times wider if the char is a multiwidth char
		// and we are dividing this width by 2. One for current cell and one for next
		atlasPos.W /= 2
		if nextIndex != -1 {
			// Draw the parts more than width to the next cell
			// NOTE: The more part has the same color with next cell
			// NOTE: Multiwidth cells causes glyphs to overlap
			secAtlasPos := common.Rectangle[int]{
				X: atlasPos.X + cellSize.Width(),
				Y: atlasPos.Y,
				W: cellSize.Width(),
				H: cellSize.Height(),
			}
			renderer.buffer.SetIndexTex2(nextIndex, renderer.atlas.Normalize(secAtlasPos))
			renderer.buffer.SetIndexFg(nextIndex, attrib.foreground)
		}
	} else if nextIndex != -1 {
		// Clear second texture.
		renderer.buffer.SetIndexTex2(nextIndex, common.ZeroRectangleF32)
	}
	// Draw
	renderer.buffer.SetIndexTex1(index, renderer.atlas.Normalize(atlasPos))
	renderer.buffer.SetIndexFg(index, attrib.foreground)
}

func (renderer *GridRenderer) Render() {
	renderer.atlas.BindTexture()
	renderer.buffer.Bind()
	renderer.buffer.Update()
	renderer.buffer.SetProjection(Editor.window.Viewport().ToF32())
	renderer.buffer.Render()
}

func (renderer *GridRenderer) Destroy() {
	renderer.atlas.Destroy()
	renderer.buffer.Destroy()
}
