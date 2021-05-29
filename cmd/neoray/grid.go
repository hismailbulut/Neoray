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
	cells       [][]Cell
	cells_ready bool
	width       int
	height      int
	default_fg  sdl.Color
	default_bg  sdl.Color
	default_sp  sdl.Color
	attributes  map[int]HighlightAttribute
}

func CreateGrid() Grid {
	return Grid{
		attributes: make(map[int]HighlightAttribute),
	}
}

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
	grid.cells_ready = true
}

func (grid *Grid) ClearCells() {
	defer measure_execution_time("Grid.ClearCells")()
	for _, row := range grid.cells {
		for _, cell := range row {
			cell.char = ""
			cell.attrib_id = 0
			cell.needs_redraw = true
		}
	}
}

func (grid *Grid) MakeAllCellsChanged() {
	for i := range grid.cells {
		for j := range grid.cells[i] {
			grid.cells[i][j].needs_redraw = true
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
		cell.needs_redraw = true
		*y++
	}
}

func (grid *Grid) Scroll(top, bot, rows, left, right int) {
	defer measure_execution_time("Grid.Scroll")()
	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ {
			copy(grid.cells[y-rows][left:right], grid.cells[y][left:right])
			EditorSingleton.renderer.CopyRowData(y-rows, y, left, right)
		}
	} else { // Scroll up, move down
		for y := (bot + rows) - 1; y >= top; y-- {
			copy(grid.cells[y-rows][left:right], grid.cells[y][left:right])
			EditorSingleton.renderer.CopyRowData(y-rows, y, left, right)
		}
	}
	// This is for cursor animation when scrolling. Simply we are moving cursor
	// with scroll area immediately, and returning back to its position with animation.
	EditorSingleton.cursor.SetPosition(EditorSingleton.cursor.X-rows, EditorSingleton.cursor.Y, true)
	EditorSingleton.cursor.SetPosition(EditorSingleton.cursor.X+rows, EditorSingleton.cursor.Y, false)
}
