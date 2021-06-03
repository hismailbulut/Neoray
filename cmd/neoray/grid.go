package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

type Cell struct {
	char         string
	attrib_id    int
	needs_redraw bool
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
	cells      [][]Cell
	width      int
	height     int
	default_fg sdl.Color
	default_bg sdl.Color
	default_sp sdl.Color
	attributes map[int]HighlightAttribute
}

func CreateGrid() Grid {
	grid := Grid{
		attributes: make(map[int]HighlightAttribute),
	}
	return grid
}

// This function is only used by neovim,
// and calling this anywhere else may break the program.
func (grid *Grid) Resize(width int, height int) {
	grid.width = width
	grid.height = height
	// row count is height, column count is width
	grid.cells = make([][]Cell, height) // rows
	for i := range grid.cells {
		grid.cells[i] = make([]Cell, width) // columns
		for j := range grid.cells[i] {
			grid.cells[i][j].needs_redraw = true
		}
	}
}

func (grid *Grid) ClearCells() {
	for _, row := range grid.cells {
		for _, cell := range row {
			cell.char = ""
			cell.attrib_id = 0
			cell.needs_redraw = true
		}
	}
}

// This makes all cells will be rendered in the next
// draw call. We are using this when a highlight attribute
// changes. Because we don't know how many and which cells
// will be affected from highlight attribute change.
func (grid *Grid) MakeAllCellsChanged() {
	for i := range grid.cells {
		for j := range grid.cells[i] {
			grid.cells[i][j].needs_redraw = true
		}
	}
}

// Sets cells with the given parameters, and advances y to the next.
// This function will not check the end of the row. And currently
// only used by neovim. If you need to set a cell for your needs,
// you should create an alternative function for your needs.
// If `repeat` is present, the cell should be
// repeated `repeat` times (including the first time)
func (grid *Grid) SetCell(x int, y *int, char string, hl_id int, repeat int) {
	cell_count := 1
	if repeat > 0 {
		cell_count = repeat
	}
	for i := 0; i < cell_count; i++ {
		cell := &grid.cells[x][*y]
		cell.char = char
		cell.attrib_id = hl_id
		cell.needs_redraw = true
		*y++
	}
}

// This function gives secure access to grid cells.
// Just check if the cell is nil or not.
func (grid *Grid) GetCell(x, y int) *Cell {
	if x < len(grid.cells) {
		if y < len(grid.cells[x]) {
			return &grid.cells[x][y]
		}
	}
	return nil
}

// TODO: Find a way to speed up.
func (grid *Grid) Scroll(top, bot, rows, left, right int) {
	defer measure_execution_time("Grid.Scroll")()
	copyCellsAndScroll := func(dst, src, left, right int) {
		copy(grid.cells[dst][left:right], grid.cells[src][left:right])
		EditorSingleton.renderer.CopyRowData(dst, src, left, right)
	}
	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ {
			copyCellsAndScroll(y-rows, y, left, right)
		}
	} else { // Scroll up, move down
		for y := (bot + rows) - 1; y >= top; y-- {
			copyCellsAndScroll(y-rows, y, left, right)
		}
	}
	// This is for cursor animation when scrolling. Simply we are moving cursor
	// with scroll area immediately, and returning back to its position smoothly.
	EditorSingleton.cursor.SetPosition(EditorSingleton.cursor.X-rows, EditorSingleton.cursor.Y, true)
	EditorSingleton.cursor.SetPosition(EditorSingleton.cursor.X+rows, EditorSingleton.cursor.Y, false)
}

// NOTE: Reserved
func (grid *Grid) Destroy() {
	log_debug_msg("Grid destroyed.")
}
