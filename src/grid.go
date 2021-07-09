package main

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
func (grid *Grid) resize(rows, cols int) {
	grid.cells = make([][]Cell, rows)
	for i := range grid.cells {
		grid.cells[i] = make([]Cell, cols)
		for j := range grid.cells[i] {
			grid.cells[i][j].needsDraw = true
		}
	}
}

func (grid *Grid) clearCells() {
	for _, row := range grid.cells {
		for _, cell := range row {
			cell.char = 0
			cell.attribId = 0
			cell.needsDraw = true
		}
	}
	EditorSingleton.draw()
}

// This makes all cells will be rendered in the next
// draw call. We are using this when a highlight attribute
// changes. Because we don't know how many and which cells
// will be affected from highlight attribute change.
func (grid *Grid) makeAllCellsChanged() {
	for i := range grid.cells {
		for j := range grid.cells[i] {
			grid.cells[i][j].needsDraw = true
		}
	}
	EditorSingleton.draw()
}

// Sets cells with the given parameters, and advances y to the next.
// This function will not check the end of the row. And currently
// only used by neovim. If you need to set a cell for your needs,
// you should create an alternative function.
// If `repeat` is present, the cell should be
// repeated `repeat` times (including the first time)
func (grid *Grid) setCells(x int, y *int, char rune, attribId int, repeat int) {
	cell_count := 1
	if repeat > 0 {
		cell_count = repeat
	}
	for i := 0; i < cell_count; i++ {
		grid.setCell(x, *y, char, attribId)
		*y++
	}
}

func (grid *Grid) setCell(x, y int, char rune, attribId int) {
	if x >= 0 && y >= 0 && x < len(grid.cells) && y < len(grid.cells[x]) {
		grid.cells[x][y].char = char
		grid.cells[x][y].attribId = attribId
		grid.cells[x][y].needsDraw = true
	} else {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Index out of bounds in setCell.")
	}
}

// This function returns a copy of the cell.
func (grid *Grid) getCell(x, y int) Cell {
	if x >= 0 && y >= 0 && x < len(grid.cells) && y < len(grid.cells[x]) {
		return grid.cells[x][y]
	} else {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Index out of bounds in getCell.")
		return Cell{}
	}
}

func (grid *Grid) copyRow(dst, src, left, right int) {
	copy(grid.cells[dst][left:right], grid.cells[src][left:right])
	EditorSingleton.renderer.copyRowData(dst, src, left, right)
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
	cursor := &EditorSingleton.cursor
	if cursor.isInArea(top, left, bot-top, right-left) {
		// This is for cursor animation when scrolling. Simply we are moving cursor
		// with scroll area immediately, and returning back to its position smoothly.
		cursor.SetPosition(cursor.X-rows, cursor.Y, true)
		cursor.SetPosition(cursor.X+rows, cursor.Y, false)
	}
	EditorSingleton.draw()
}
