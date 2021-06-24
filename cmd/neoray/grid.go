package main

type Cell struct {
	char         string
	attrib_id    int
	needs_redraw bool
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

type Grid struct {
	cells      [][]Cell
	default_fg U8Color
	default_bg U8Color
	default_sp U8Color
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
func (grid *Grid) Resize(rows, cols int) {
	grid.cells = make([][]Cell, rows)
	for i := range grid.cells {
		grid.cells[i] = make([]Cell, cols)
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
// you should create an alternative function.
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

// This function returns a copy of the cell.
func (grid *Grid) GetCell(x, y int) Cell {
	if x < len(grid.cells) && y < len(grid.cells[x]) {
		return grid.cells[x][y]
	}
	return Cell{}
}

func (grid *Grid) Scroll(top, bot, rows, left, right int) {
	defer measure_execution_time("Grid.Scroll")()
	copyCellsAndScroll := func(dst, src, left, right int) {
		copy(grid.cells[dst][left:right], grid.cells[src][left:right])
		EditorSingleton.renderer.copyRowData(dst, src, left, right)
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
	cursor := &EditorSingleton.cursor
	if cursor.isInArea(top, left, bot-top, right-left) {
		// This is for cursor animation when scrolling. Simply we are moving cursor
		// with scroll area immediately, and returning back to its position smoothly.
		cursor.SetPosition(cursor.X-rows, cursor.Y, true)
		cursor.SetPosition(cursor.X+rows, cursor.Y, false)
	}
}
