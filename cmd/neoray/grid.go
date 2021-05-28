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
	cells             [][]Cell
	cells_ready       bool
	width             int
	height            int
	default_fg        sdl.Color
	default_bg        sdl.Color
	default_sp        sdl.Color
	attributes        map[int]HighlightAttribute
	scroll_anim       Animation
	cell_change_queue map[ivec2]Cell
}

func CreateGrid() Grid {
	return Grid{
		attributes:        make(map[int]HighlightAttribute),
		cell_change_queue: make(map[ivec2]Cell),
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
		// We will queue changes if renderer is busy with animations
		if EditorSingleton.renderer.scroll_info.needs_redraw {
			grid.cell_change_queue[ivec2{X: x, Y: *y}] = Cell{
				char:         char,
				attrib_id:    hl_id,
				needs_redraw: true,
			}
		} else {
			cell := &grid.cells[x][*y]
			cell.char = char
			cell.attrib_id = hl_id
			cell.needs_redraw = true
		}
		*y++
	}
}

func (grid *Grid) ApplyCellChanges() {
	if len(grid.cell_change_queue) > 0 {
		for key, val := range grid.cell_change_queue {
			grid.cells[key.X][key.Y] = val
			delete(grid.cell_change_queue, key)
		}
		// We need to flush renderer because neovim already sends it's flush.
		EditorSingleton.renderer.DrawAllChangedCells()
	}
}

// This is used both grid and renderer functions and every loop is same.
// I have just done it like that because I'm obsessed with this kind of things.
// Function is called in every row and does not include left and right borders
// of scroll area. User should exclude left and right borders in function.
func (grid *Grid) ScrollIterator(top, bot, rows int,
	doPerIteration func(dst_row, src_row int)) {
	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ {
			doPerIteration(y-rows, y)
		}
	} else { // Scroll up, move down
		for y := (bot + rows) - 1; y >= top; y-- {
			doPerIteration(y-rows, y)
		}
	}
}

func (grid *Grid) Scroll(top, bot, rows, left, right int) {
	defer measure_execution_time("Grid.Scroll")()
	EditorSingleton.renderer.BeginScrolling(top, bot, rows, left, right)
	grid.ScrollIterator(top, bot, rows,
		func(dst_row, src_row int) {
			copy(grid.cells[dst_row][left:right], grid.cells[src_row][left:right])
		})
}
