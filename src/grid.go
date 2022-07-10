package main

import (
	"fmt"

	"github.com/hismailbulut/neoray/src/bench"
	"github.com/hismailbulut/neoray/src/common"
	"github.com/hismailbulut/neoray/src/fontkit"
	"github.com/hismailbulut/neoray/src/logger"
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
// Current size of the Cell is 16 bytes in 64 bit systems
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
	return fmt.Sprintf("Grid(ID: %d, Number: %d, Type: %s, Pos: %d %d, PixelPos: %v, Size: %d %d, Window: %d, Hidden: %t)",
		grid.id,
		grid.number,
		grid.typ,
		grid.sRow, grid.sCol,
		grid.renderer.position,
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
	grid.renderer, err = NewGridRenderer(window, rows, cols, kit, fontSize, position)
	if err != nil {
		return nil, err
	}
	logger.Log(logger.DEBUG, "Grid created:", grid)
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
	MarkRender()
}

// Don't use this function directly. Use gridManager's resize function.
func (grid *Grid) Resize(rows, cols int) {
	// Don't resize if size is already same
	if rows == grid.rows && cols == grid.cols {
		return
	}
	// TODO: We can reduce allocations by making cells 1 dimensional array
	// NOTE: Resizing should not clear the cells
	EndBenchmark := bench.BeginBenchmark()
	// Resize rows
	if cap(grid.cells) > rows {
		grid.cells = grid.cells[:rows]
	} else {
		remaining := rows - len(grid.cells)
		grid.cells = append(grid.cells, make([][]Cell, remaining)...)
	}
	// Resize cols
	for i := 0; i < rows; i++ {
		if cap(grid.cells[i]) > cols {
			grid.cells[i] = grid.cells[i][:cols]
		} else {
			remaining := cols - len(grid.cells[i])
			grid.cells[i] = append(grid.cells[i], make([]Cell, remaining)...)
		}
	}
	EndBenchmark("Grid.ResizeCells")
	// Resize renderer
	grid.renderer.Resize(rows, cols)
	grid.rows = rows
	grid.cols = cols
	// Resizing renderer also clears it's buffer, because of this we must redraw every cell
	MarkForceDraw()
}

func (grid *Grid) SetPos(win, sRow, sCol int, rows, cols int, typ GridType, position common.Vector2[int]) {
	grid.window = win
	grid.typ = typ
	grid.hidden = false
	grid.sRow = sRow
	grid.sCol = sCol
	// NOTE: I don't know if this is required
	// grid.Resize(rows, cols)
	grid.renderer.SetPos(position)
	logger.Log(logger.DEBUG, "Grid moved:", grid)
	MarkForceDraw()
}

func (grid *Grid) Draw(force bool) {
	if grid.hidden {
		return
	}
	EndBenchmark := bench.BeginBenchmark()
	for row := 0; row < grid.rows; row++ {
		for col := 0; col < grid.cols; col++ {
			cell := grid.CellAt(row, col)
			if force || cell.needsDraw {
				attrib := Editor.gridManager.Attribute(cell.attribID)
				// Override background alpha
				// TODO We must check whether it is default background or not
				attrib.background.A = uint8(Editor.options.transparency * 255)
				grid.renderer.DrawCell(row, col, cell.char, attrib)
				cell.needsDraw = false
			}
		}
	}
	if force {
		EndBenchmark("Grid.ForceDraw")
	} else {
		EndBenchmark("Grid.Draw")
	}
}

func (grid *Grid) Render() {
	if grid.hidden {
		return
	}
	grid.renderer.Render()
}

func (grid *Grid) Destroy() {
	logger.Log(logger.DEBUG, "Grid destroyed:", grid)
	grid.renderer.Destroy()
}
