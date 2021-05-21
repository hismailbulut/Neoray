package main

import "github.com/veandco/go-sdl2/sdl"

type Cell struct {
	char      string
	attrib_id int
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

func (table *Grid) Resize(width int, height int) {
	table.width = width
	table.height = height
	// row count is height, column count is width
	table.cells = make([][]Cell, height) // rows
	for i := range table.cells {
		table.cells[i] = make([]Cell, width) // columns
		table.changed_rows[i] = true
	}

}

func (table *Grid) ClearCells() {
	for i, row := range table.cells {
		for _, cell := range row {
			cell.char = ""
			cell.attrib_id = 0
		}
		table.changed_rows[i] = true
	}
}

func (grid *Grid) MakeAllCellsChanged() {
	for i := range grid.cells {
		grid.changed_rows[i] = true
	}
}

func (table *Grid) SetCell(x int, y *int, char string, hl_id int, repeat int) {
	// If `repeat` is present, the cell should be
	// repeated `repeat` times (including the first time)
	cell_count := 1
	if repeat > 0 {
		cell_count = repeat
	}
	for i := 0; i < cell_count; i++ {
		cell := &table.cells[x][*y]
		cell.char = char
		cell.attrib_id = hl_id
		*y++
	}
	table.changed_rows[x] = true
}

func (table *Grid) Scroll(top, bot, rows, left, right int) {
	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ { // row
			copy(table.cells[y-rows][left:right], table.cells[y][left:right])
			table.changed_rows[y-rows] = true
		}
	} else { // Scroll up, move down
		// rows is negative
		for y := (bot + rows) - 1; y >= top; y-- { // row
			copy(table.cells[y-rows][left:right], table.cells[y][left:right])
			table.changed_rows[y-rows] = true
		}
	}
}
