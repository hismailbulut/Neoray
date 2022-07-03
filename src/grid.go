package main

import (
	"fmt"

	"github.com/hismailbulut/neoray/src/common"
	"github.com/hismailbulut/neoray/src/fontkit"
	"github.com/hismailbulut/neoray/src/window"
)

type GridType int32

const (
	GridTypeNormal  GridType = iota // Normal grid
	GridTypeMessage                 // Message grid, will be rendered front of the normal grids
	GridTypeFloat                   // Float window, will be rendered most front
)

func (gridType GridType) String() string {
	switch gridType {
	case GridTypeNormal:
		return "Normal"
	case GridTypeMessage:
		return "Message"
	case GridTypeFloat:
		return "Float"
	}
	panic("unknown grid type")
}

// Because of the struct packing, this order of elements is important
// Current size of the Cell is 16 bytes
type Cell struct {
	needsDraw bool
	char      rune
	attribID  int
}

func (cell Cell) String() string {
	return fmt.Sprintf("Cell(Char: %d %s, AttribID: %d)",
		cell.char,
		string(cell.char),
		cell.attribID,
	)
}

type Grid struct {
	id         int // id is the same id used in the grids hashmap
	number     int // number specifies the create order of the grid, which starts from zero and counts
	sRow, sCol int // top left corner of the grid
	rows, cols int // rows and columns of the grid
	window     int // grid's window id
	hidden     bool
	typ        GridType
	renderer   *GridRenderer
	cells      [][]Cell
}

// For debugging purposes.
func (grid *Grid) String() string {
	return fmt.Sprintf("Grid(ID: %d, Number: %d, Type: %s, Pos: %d %d, Size: %d %d, WinID: %d, Hidden: %t)",
		grid.id,
		grid.number,
		grid.typ,
		grid.sRow, grid.sCol,
		grid.rows, grid.cols,
		grid.window,
		grid.hidden,
	)
}

func NewGrid(window *window.Window, id, number int, rows, cols int, kit *fontkit.FontKit, fontSize float64, position common.Vector2[int]) (*Grid, error) {
	grid := new(Grid)
	grid.id = id
	grid.number = number
	grid.rows = rows
	grid.cols = cols
	// Create cells
	grid.cells = make([][]Cell, rows)
	for i := range grid.cells {
		grid.cells[i] = make([]Cell, cols)
	}
	// Create renderer
	var err error
	grid.renderer, err = NewGridRenderer(window, grid, rows, cols, kit, fontSize, position)
	if err != nil {
		return nil, err
	}
	return grid, nil
}

// Font related

func (grid *Grid) SetFontKit(kit *fontkit.FontKit) {
	grid.renderer.SetFontKit(kit)
}

func (grid *Grid) SetFontSize(fontSize, dpi float64) {
	grid.renderer.SetFontSize(fontSize, dpi)
}

func (grid *Grid) AddFontSize(v, dpi float64) {
	fontSize := grid.renderer.FontSize() + v
	grid.renderer.SetFontSize(fontSize, dpi)
}

func (grid *Grid) SetBoxDrawing(useBoxDrawing, useBlockDrawing bool) {
	grid.renderer.SetBoxDrawing(useBoxDrawing, useBlockDrawing)
}

func (grid *Grid) Size() common.Vector2[int] {
	return common.Vector2[int]{
		X: grid.cols * grid.CellSize().Width(),
		Y: grid.rows * grid.CellSize().Height(),
	}
}

// This function returns a copy of the cell. Does not check bounds.
func (grid *Grid) CellAt(row, col int) Cell {
	return grid.cells[row][col]
}

// Sets the cell in grid. Does not check bounds.
// Only internal usage
func (grid *Grid) SetCell(row, col int, char rune, attribID int) {
	grid.cells[row][col].char = char
	grid.cells[row][col].attribID = attribID
	grid.cells[row][col].needsDraw = true
}

func (grid *Grid) PixelPos() common.Vector2[int] {
	return grid.renderer.position
}

func (grid *Grid) CellSize() common.Vector2[int] {
	return grid.renderer.cellSize
}

func (grid *Grid) Scroll(top, bot, rows, left, right int) {
	// dst and src are row numbers
	// left and right are column numbers
	copyRow := func(dst, src, left, right int) {
		copy(grid.cells[dst][left:right], grid.cells[src][left:right])
		grid.renderer.CopyRow(dst, src, left, right)
		// singleton.renderer.copyRowData(dst+grid.sRow, src+grid.sRow, left+grid.sCol, right+grid.sCol)
	}

	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ {
			copyRow(y-rows, y, left, right)
		}
	} else { // Scroll up, move down
		for y := (bot + rows) - 1; y >= top; y-- {
			copyRow(y-rows, y, left, right)
		}
	}

	// Animate cursor when scrolling
	if Editor.cursor.IsInArea(grid.id, top, left, bot-top, right-left) {
		// This is for cursor animation when scrolling. Simply we are moving cursor
		// with scroll area immediately, and returning back to its position smoothly.
		targetRow := Editor.cursor.row - rows
		cursorGrid := Editor.cursor.Grid()
		if cursorGrid != nil {
			if targetRow >= 0 && targetRow < cursorGrid.rows {
				currentRow := Editor.cursor.row
				Editor.cursor.SetPosition(Editor.cursor.grid, targetRow, Editor.cursor.col, true)
				Editor.cursor.SetPosition(Editor.cursor.grid, currentRow, Editor.cursor.col, false)
			}
		}
	}

	MarkRender()
}

// Don't use this function directly. Use gridManager's resize function.
func (grid *Grid) Resize(rows, cols int) {
	// Don't resize if size is already same
	if rows == grid.rows && cols == grid.cols {
		return
	}
	// TODO: make cells 1 dimension
	// Resizing should not clear the cells
	// Resize rows
	if cap(grid.cells) > rows {
		grid.cells = grid.cells[:rows]
	} else {
		remaining := rows - len(grid.cells)
		grid.cells = append(grid.cells, make([][]Cell, remaining)...)
	}
	assert(len(grid.cells) == rows)
	// Resize cols
	for i := 0; i < rows; i++ {
		if cap(grid.cells[i]) > cols {
			grid.cells[i] = grid.cells[i][:cols]
		} else {
			remaining := cols - len(grid.cells[i])
			grid.cells[i] = append(grid.cells[i], make([]Cell, remaining)...)
		}
		assert(len(grid.cells[i]) == cols)
	}
	// Resize renderer
	grid.renderer.Resize(rows, cols)
	grid.rows = rows
	grid.cols = cols
	MarkForceDraw()
}

func (grid *Grid) SetPos(win, sRow, sCol int, rows, cols int, typ GridType, position common.Vector2[int]) {
	grid.window = win
	grid.typ = typ
	grid.hidden = false

	grid.sRow = sRow
	grid.sCol = sCol
	// grid.Resize(rows, cols)

	grid.renderer.SetPos(position)

	MarkForceDraw()
}

func (grid *Grid) Draw(force bool) {
	if grid.hidden {
		return
	}
	for row := 0; row < grid.rows; row++ {
		for col := 0; col < grid.cols; col++ {
			cell := grid.CellAt(row, col)
			if force || cell.needsDraw {
				grid.renderer.DrawCell(row, col, cell)
				cell.needsDraw = false
			}
		}
	}
}

func (grid *Grid) Render() {
	if grid.hidden {
		return
	}
	grid.renderer.Render()
}

func (grid *Grid) Destroy() {
	grid.renderer.Destroy()
}
