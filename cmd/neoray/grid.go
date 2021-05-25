package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

type Cell struct {
	char      string
	attrib_id int
	changed   bool
}

type HighlightAttribute struct {
	foreground    sdl.Color
	background    sdl.Color
	special       sdl.Color
	reverse       bool
	italic        bool
	bold          bool
	strikethrough bool
	underline     bool
	undercurl     bool
	blend         int
}

type Grid struct {
	cells        [][]Cell
	cells_ready  bool
	width        int
	height       int
	default_fg   sdl.Color
	default_bg   sdl.Color
	default_sp   sdl.Color
	attributes   map[int]HighlightAttribute
	changed_rows map[int]bool
}

func CreateGrid() Grid {
	return Grid{
		attributes:   make(map[int]HighlightAttribute),
		changed_rows: make(map[int]bool),
	}
}

func (grid *Grid) Resize(width int, height int) {
	grid.width = width
	grid.height = height
	// row count is height, column count is width
	grid.cells = make([][]Cell, height) // rows
	for i := range grid.cells {
		grid.cells[i] = make([]Cell, width) // columns
		grid.changed_rows[i] = true
		for j := range grid.cells[i] {
			grid.cells[i][j].changed = true
		}
	}
	grid.cells_ready = true
}

func (grid *Grid) ClearCells() {
	for i, row := range grid.cells {
		for _, cell := range row {
			cell.char = ""
			cell.attrib_id = 0
			cell.changed = true
		}
		grid.changed_rows[i] = true
	}
}

func (grid *Grid) MakeAllCellsChanged() {
	for i := range grid.cells {
		grid.changed_rows[i] = true
		for j := range grid.cells[i] {
			grid.cells[i][j].changed = true
		}
	}
}

func (grid *Grid) SetCell(x int, y *int, char string, hl_id int, repeat int) {
	// If `repeat` is present, the cell should be
	// repeated `repeat` times (including the first time)
	cell_count := 1
	if repeat > 0 {
		cell_count = repeat
	}
	for i := 0; i < cell_count; i++ {
		cell := &grid.cells[x][*y]
		cell.char = char
		cell.attrib_id = hl_id
		cell.changed = true
		*y++
	}
	grid.changed_rows[x] = true
}

func (grid *Grid) Scroll(top, bot, rows, left, right int, renderer *Renderer) {
	defer measure_execution_time("Grid.Scroll")()
	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ { // row
			copy(grid.cells[y-rows][left:right], grid.cells[y][left:right])
			renderer.CopyRowData(y-rows, y, left, right)
		}
	} else { // Scroll up, move down
		// rows is negative
		for y := (bot + rows) - 1; y >= top; y-- { // row
			copy(grid.cells[y-rows][left:right], grid.cells[y][left:right])
			renderer.CopyRowData(y-rows, y, left, right)
		}
	}
}
