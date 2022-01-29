package main

import (
	"fmt"
	"sort"
)

type Cell struct {
	char      rune
	attribId  int
	needsDraw bool
}

type HighlightAttribute struct {
	foreground    U8Color
	background    U8Color
	special       U8Color
	reverse       bool
	italic        bool
	bold          bool
	strikethrough bool
	underline     bool
	undercurl     bool
	blend         int
}

type GridType int32

const (
	GridTypeNormal  GridType = iota // Normal grid
	GridTypeMessage                 // Message grid, will be rendered front of the normal grids
	GridTypeFloat                   // Float window, will be rendered most front
)

type Grid struct {
	id         int // id is the same id used in the grids hashmap
	number     int // number specifies the create order of the grid, which starts from zero and counts
	typ        GridType
	sRow, sCol int // top left corner of the grid
	rows, cols int // rows and columns of the grid
	window     int // grid's window id
	hidden     bool
	cells      [][]Cell
}

type GridManager struct {
	grids       map[int]*Grid
	attributes  map[int]HighlightAttribute
	defaultFg   U8Color
	defaultBg   U8Color
	defaultSp   U8Color
	sortedGrids []*Grid
}

func CreateGridManager() GridManager {
	grid := GridManager{
		grids:      make(map[int]*Grid),
		attributes: make(map[int]HighlightAttribute),
	}
	return grid
}

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

// For debugging purposes.
func (grid *Grid) String() string {
	return fmt.Sprint("Id: ", grid.id, " Nr: ", grid.number,
		" Y: ", grid.sRow, " X: ", grid.sCol, " H: ", grid.rows, " W: ", grid.cols,
		" Win: ", grid.window, " Hidden: ", grid.hidden, " Type: ", grid.typ)
}

// Sorts grids according to rendering order and returns it.
// You can access the sorted array via gridManager.sortedGrids
// and don't call this function directly.
func (gridManager *GridManager) sortGrids() []*Grid {
	// Resize sorted slice to length of the grids slice
	if len(gridManager.sortedGrids) < len(gridManager.grids) {
		gridManager.sortedGrids = make([]*Grid, len(gridManager.grids))
	} else {
		gridManager.sortedGrids = gridManager.sortedGrids[:len(gridManager.grids)]
	}
	// Copy grids to slice
	i := 0
	for _, grid := range gridManager.grids {
		gridManager.sortedGrids[i] = grid
		i++
	}
	if len(gridManager.sortedGrids) > 1 {
		// Sort
		sort.Slice(gridManager.sortedGrids,
			func(i, j int) bool {
				g1 := gridManager.sortedGrids[i]
				g2 := gridManager.sortedGrids[j]
				if g1.typ > g2.typ {
					return false
				}
				if g1.typ < g2.typ {
					return true
				}
				return g1.number < g2.number
			})
	}
	return gridManager.sortedGrids
}

// Returns grid id and cell position at the given global position.
// The returned values are grid id, cell row, cell column
func (gridManager *GridManager) getCellAt(pos IntVec2) (int, int, int) {
	// The input_mouse api call wants 0 for grid when multigrid is not enabled
	if singleton.parsedArgs.multiGrid == false {
		return 0, pos.Y / singleton.cellHeight, pos.X / singleton.cellWidth
	}
	id, row, col := -1, -1, -1
	// Find top grid at this position
	for i := len(gridManager.sortedGrids) - 1; i >= 0; i-- {
		grid := gridManager.sortedGrids[i]
		if !grid.hidden {
			gridRect := IntRect{
				X: grid.sCol * singleton.cellWidth,
				Y: grid.sRow * singleton.cellHeight,
				W: grid.cols * singleton.cellWidth,
				H: grid.rows * singleton.cellHeight,
			}
			if pos.inRect(gridRect) {
				id = grid.id
				// Calculate cell position
				row = (pos.Y - gridRect.Y) / singleton.cellHeight
				col = (pos.X - gridRect.X) / singleton.cellWidth
				break
			}
		}
	}
	return id, row, col
}

func (gridManager *GridManager) resize(id int, rows, cols int) {
	grid, ok := gridManager.grids[id]
	if !ok {
		grid = new(Grid)
		grid.id = id
		grid.number = len(gridManager.grids)
		gridManager.grids[id] = grid
	}
	grid.resize(rows, cols)
}

// Don't use this function directly. Use gridManager's resize function.
func (grid *Grid) resize(rows, cols int) {
	// Don't resize if size is already same
	if rows == grid.rows && cols == grid.cols {
		return
	}
	// Resize grid and copy cells
	if len(grid.cells) < rows {
		temp := make([][]Cell, len(grid.cells))
		copy(temp, grid.cells)
		grid.cells = make([][]Cell, rows)
		copy(grid.cells, temp)
	} else {
		grid.cells = grid.cells[:rows]
	}
	for i := 0; i < rows; i++ {
		if len(grid.cells[i]) < cols {
			temp := make([]Cell, len(grid.cells[i]))
			copy(temp, grid.cells[i])
			grid.cells[i] = make([]Cell, cols)
			copy(grid.cells[i], temp)
		} else {
			grid.cells[i] = grid.cells[i][:cols]
		}
	}
	grid.rows = rows
	grid.cols = cols
	singleton.fullDraw()
}

func (grid *Grid) setPos(win, sRow, sCol, rows, cols int, typ GridType) {
	grid.window = win
	grid.typ = typ
	grid.hidden = false

	grid.sRow = sRow
	grid.sCol = sCol
	grid.resize(rows, cols)

	singleton.fullDraw()
}

func (gridManager *GridManager) hide(id int) {
	grid, ok := gridManager.grids[id]
	if ok {
		grid.hidden = true
		// NOTE: Hide and destroy functions are only calling when multigrid is on.
		// When this functions called from neovim, we know which grid is hided or
		// destroyed but we dont know how many grids affected. Because grids can
		// overlap and hiding a grid on top of a grid causes back grid needs to be
		// rendered. This is also applies to setPos. We could also try to detect which
		// grid must be drawed but fully drawing screen is fast and more stable.
		singleton.fullDraw()
	}
}

func (gridManager *GridManager) destroy(id int) {
	_, ok := gridManager.grids[id]
	if ok {
		delete(gridManager.grids, id)
		singleton.fullDraw()
	}
}

func (gridManager *GridManager) clear(id int) {
	grid, ok := gridManager.grids[id]
	if ok {
		for i := 0; i < grid.rows; i++ {
			for j := 0; j < grid.cols; j++ {
				grid.cells[i][j].char = 0
				grid.cells[i][j].attribId = 0
				grid.cells[i][j].needsDraw = true
			}
		}
		singleton.draw()
	}
}

// Sets cells with the given parameters, and advances y to the next. If
// `repeat` is present, the cell should be repeated `repeat` times (including
// the first time). This function will not check the end of the row. And
// currently only used by neovim.
func (gridManager *GridManager) setCell(id, x int, y *int, char rune, attribId, repeat int) {
	grid, ok := gridManager.grids[id]
	if ok {
		cell_count := 1
		if repeat > 0 {
			cell_count = repeat
		}
		for i := 0; i < cell_count; i++ {
			grid.setCell(x, *y, char, attribId)
			*y++
		}
	}
}

// Sets the cell in grid. Does not check bounds.
func (grid *Grid) setCell(x, y int, char rune, attribId int) {
	grid.cells[x][y].char = char
	grid.cells[x][y].attribId = attribId
	grid.cells[x][y].needsDraw = true
}

// This function returns a copy of the cell. Does not check bounds.
func (grid *Grid) getCell(x, y int) Cell {
	return grid.cells[x][y]
}

func (grid *Grid) copyRow(dst, src, left, right int) {
	copy(grid.cells[dst][left:right], grid.cells[src][left:right])
	// Renderer needs global position
	singleton.renderer.copyRowData(dst+grid.sRow, src+grid.sRow, left+grid.sCol, right+grid.sCol)
}

func (grid *Grid) scroll(top, bot, rows, left, right int) {
	defer measure_execution_time()()
	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ {
			grid.copyRow(y-rows, y, left, right)
		}
	} else { // Scroll up, move down
		for y := (bot + rows) - 1; y >= top; y-- {
			grid.copyRow(y-rows, y, left, right)
		}
	}
	// Animate cursor when scrolling
	cursor := &singleton.cursor
	if cursor.isInArea(grid.id, top, left, bot-top, right-left) {
		// This is for cursor animation when scrolling. Simply we are moving cursor
		// with scroll area immediately, and returning back to its position smoothly.
		target := cursor.X - rows
		if target >= 0 && target < singleton.gridManager.grids[cursor.grid].rows {
			current := cursor.X
			cursor.setPosition(cursor.grid, target, cursor.Y, true)
			cursor.setPosition(cursor.grid, current, cursor.Y, false)
		}
	}
	// We dont need to draw screen because we already directly moved vertex
	// data. Only rendering will be fine.
	singleton.render()
}
