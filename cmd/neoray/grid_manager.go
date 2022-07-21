package main

import (
	"sort"

	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/fontkit"
	"github.com/hismailbulut/Neoray/pkg/logger"
)

type GridManager struct {
	grids       map[int]*Grid
	sortedGrids []*Grid
	// These are used for creating new grids
	totalGridsCreated int              // total number of grids created (including deleted ones)
	kit               *fontkit.FontKit // last globally set font kit
	fontSize          float64          // last globally set font size
	// style information
	attributes map[int]HighlightAttribute
	foreground common.Color[uint8] // Default foreground color
	background common.Color[uint8] // Default background color
	special    common.Color[uint8] // Default special color
}

func NewGridManager() *GridManager {
	grid := &GridManager{
		grids:      make(map[int]*Grid),
		attributes: make(map[int]HighlightAttribute),
	}
	return grid
}

// Attribute itself reverses the color if required, so you dont have to swap them
// for rendering grids
func (manager *GridManager) Attribute(id int) (attrib HighlightAttribute) {
	var ok bool
	if id == 0 {
		// Default attribute
		attrib.foreground = manager.foreground
		attrib.background = manager.background
		attrib.special = manager.special
	} else {
		attrib, ok = manager.attributes[id]
		if !ok {
			logger.LogF(logger.ERROR, "Attribute id %d not found!", id)
		}
		// Zero alpha means color is not set yet and we use default color
		if attrib.foreground.A == 0 {
			attrib.foreground = manager.foreground
		}
		if attrib.background.A == 0 {
			attrib.background = manager.background
		}
		if attrib.special.A == 0 {
			attrib.special = manager.special
		}
		// Reverse foreground an background colors if reverse attribute set
		if attrib.reverse {
			attrib.foreground, attrib.background = attrib.background, attrib.foreground
		}
	}
	return attrib
}

// Font related

func (manager *GridManager) SetGridFontKit(id int, kit *fontkit.FontKit) {
	if id == 1 {
		for _, grid := range manager.grids {
			grid.SetFontKit(kit)
		}
		manager.kit = kit
		manager.CheckDefaultGridSize()
	} else {
		grid := manager.Grid(id)
		if grid != nil {
			prevSize := grid.Size()
			grid.SetFontKit(kit)
			manager.CheckGridSize(grid, prevSize)
		}
	}
	MarkForceDraw()
}

func (manager *GridManager) ResetFontSize() {
	for _, grid := range manager.grids {
		grid.SetFontSize(grid.renderer.FontSize(), Editor.window.DPI())
	}
	manager.CheckDefaultGridSize()
}

func (manager *GridManager) SetGridFontSize(id int, fontSize float64) {
	if id == 1 {
		for _, grid := range manager.grids {
			grid.SetFontSize(fontSize, Editor.window.DPI())
		}
		manager.fontSize = fontSize
		manager.CheckDefaultGridSize()
	} else {
		grid := manager.Grid(id)
		if grid != nil {
			prevSize := grid.Size()
			grid.SetFontSize(fontSize, Editor.window.DPI())
			manager.CheckGridSize(grid, prevSize)
		}
	}
	MarkForceDraw()
}

func (manager *GridManager) AddGridFontSize(id int, v float64) {
	if id == 1 {
		for _, grid := range manager.grids {
			grid.AddFontSize(v, Editor.window.DPI())
		}
		manager.fontSize += v
		manager.CheckDefaultGridSize()
	} else {
		grid := manager.Grid(id)
		if grid != nil {
			prevSize := grid.Size()
			grid.AddFontSize(v, Editor.window.DPI())
			manager.CheckGridSize(grid, prevSize)
		}
	}
	MarkForceDraw()
}

func (manager *GridManager) SetBoxDrawing(useBoxDrawing, useBlockDrawing bool) {
	for _, grid := range manager.grids {
		grid.SetBoxDrawing(useBoxDrawing, useBlockDrawing)
	}
	MarkForceDraw()
}

func (manager *GridManager) CheckDefaultGridSize() {
	// We should resize the default grid after font or fontsize change because cell size may has changed
	defaultGrid := manager.Grid(1)
	if defaultGrid != nil {
		cols := Editor.window.Size().Width() / defaultGrid.CellSize().Width()
		rows := Editor.window.Size().Height() / defaultGrid.CellSize().Height()
		if rows != defaultGrid.rows || cols != defaultGrid.cols {
			Editor.nvim.tryResizeUI(rows, cols)
		}
	}
}

func (manager *GridManager) CheckGridSize(grid *Grid, prevSize common.Vector2[int]) {
	// NOTE: NOT TESTED WELL
	rows := prevSize.Height() / grid.CellSize().Height()
	cols := prevSize.Width() / grid.CellSize().Width()
	if rows != grid.rows || cols != grid.cols {
		// After resizing a grid independently, neovim gives responsibility
		// of this grid to the ui. So we must mark this grid and keep it's size.
		// There is two thing we must handle
		// First when another grid resized, we should look if our grid affected by this resize
		// Second when the window size is changed, this is actually can be solved by solving first
		// But neovim gives us every grid, we must look it as a separate problem
		// TODO: per grid font size is not supported at this time and this function is not called anywhere
		// but we designed Neoray to support per grid font and size and it will be implemented in the future
		Editor.nvim.tryResizeGrid(grid.id, rows, cols)
	}
}

// Sorts grids according to rendering order and returns it.
// You can access the sorted array via gridManager.sortedGrids
// and don't call this function directly.
func (manager *GridManager) SortGrids() {
	// Resize sorted slice to length of the grids slice
	if len(manager.grids) == 0 {
		return
	}
	if len(manager.sortedGrids) != len(manager.grids) {
		manager.sortedGrids = make([]*Grid, len(manager.grids))
	}
	// Copy grids to slice
	i := 0
	for _, grid := range manager.grids {
		manager.sortedGrids[i] = grid
		i++
	}
	// Sort
	if len(manager.sortedGrids) > 1 {
		sort.Slice(manager.sortedGrids, func(i, j int) bool {
			g1 := manager.sortedGrids[i]
			g2 := manager.sortedGrids[j]
			if g1.typ > g2.typ {
				return false
			}
			if g1.typ < g2.typ {
				return true
			}
			return g1.number < g2.number
		})
	}
}

// Returns grid id and cell position at the given global position.
// The returned values are grid id, cell row, cell column
// Returned grid is always 1 if multigrid is off
func (manager *GridManager) CellAt(pos common.Vector2[int]) (int, int, int) {
	id, row, col := -1, -1, -1
	// The input_mouse api call wants 0 for grid when multigrid is not enabled
	if Editor.parsedArgs.multiGrid == false {
		// get cell size of the global grid
		defaultGrid := manager.Grid(1)
		if defaultGrid != nil {
			cellSize := defaultGrid.CellSize()
			id = 1
			row = pos.Y / cellSize.Height()
			col = pos.X / cellSize.Width()
		}
	} else {
		// Multigrid enabled
		for _, grid := range manager.sortedGrids {
			if grid.hidden {
				continue
			}
			gridPos := manager.GridPosition(grid.sRow, grid.sCol)
			gridRect := common.Rectangle[int]{
				X: gridPos.X,
				Y: gridPos.Y,
				W: grid.cols * grid.CellSize().Width(),
				H: grid.rows * grid.CellSize().Height(),
			}
			if pos.IsInRect(gridRect) {
				id = grid.id
				row = (pos.Y - gridPos.Y) / grid.CellSize().Height()
				col = (pos.X - gridPos.X) / grid.CellSize().Width()
				break
			}
		}
	}
	return id, row, col
}

// For debugging
func (manager *GridManager) printCellInfoAt(pos common.Vector2[int]) {
	gridID, row, col := manager.CellAt(pos)
	grid := manager.Grid(gridID)
	if grid != nil {
		cell := grid.CellAt(row, col)
		vertex := grid.renderer.CellVertexData(row, col)
		logger.LogF(logger.DEBUG, "CellInfo (%d, %d, %d): %v, %v, %v",
			gridID, row, col,
			grid, cell, vertex,
		)
	}
}

// Calculates and returns pixel position of a grid
func (manager *GridManager) GridPosition(sRow, sCol int) common.Vector2[int] {
	position := common.Vec2(0, 0)
	defaultGrid := manager.Grid(1)
	if defaultGrid != nil {
		position.X = sCol * defaultGrid.CellSize().Width()
		position.Y = sRow * defaultGrid.CellSize().Height()
	}
	return position
}

func (manager *GridManager) SetGridPos(id, win, sRow, sCol, rows, cols int, typ GridType) {
	grid, ok := manager.grids[id]
	if ok {
		position := manager.GridPosition(sRow, sCol)
		grid.SetPos(win, sRow, sCol, rows, cols, typ, position)
	}
}

func (manager *GridManager) Grid(id int) *Grid {
	grid, ok := manager.grids[id]
	if ok {
		return grid
	}
	return nil
}

func (manager *GridManager) ResizeGrid(id int, rows, cols int) {
	grid, ok := manager.grids[id]
	if ok {
		grid.Resize(rows, cols)
	} else {
		// Create new grid with this id
		manager.totalGridsCreated++
		if manager.fontSize == 0 {
			manager.fontSize = DEFAULT_FONT_SIZE
		}
		var err error
		grid, err = NewGrid(Editor.window, id, manager.totalGridsCreated, rows, cols, manager.kit, manager.fontSize, common.Vec2(0, 0))
		if err != nil {
			logger.Log(logger.FATAL, "Grid creation failed:", err)
		}
		manager.grids[id] = grid
	}
	MarkForceDraw()
}

func (manager *GridManager) ScrollGrid(id, top, bot, rows, left, right int) {
	grid, ok := manager.grids[id]
	if ok {
		grid.Scroll(top, bot, rows, left, right)
	}
}

func (manager *GridManager) HideGrid(id int) {
	grid, ok := manager.grids[id]
	if ok {
		grid.hidden = true
		// NOTE: Hide and destroy functions are only calling when multigrid is on.
		// When this functions called from neovim, we know which grid is hided or
		// destroyed but we dont know how many grids affected. Because grids can
		// overlap and hiding a grid on top of a grid causes back grid needs to be
		// rendered. This is also applies to setPos. We could also try to detect which
		// grid must be drawed but fully drawing screen is fast and more stable.
		MarkForceDraw()
	}
}

func (manager *GridManager) DestroyGrid(id int) {
	grid, ok := manager.grids[id]
	if ok {
		delete(manager.grids, id)
		grid.Destroy()
		MarkForceDraw()
	}
}

func (manager *GridManager) Destroy() {
	for k := range manager.grids {
		manager.DestroyGrid(k)
	}
	logger.Log(logger.DEBUG, "Grid manager destroyed")
}

func (manager *GridManager) ClearGrid(id int) {
	grid, ok := manager.grids[id]
	if ok {
		for row := 0; row < grid.rows; row++ {
			for col := 0; col < grid.cols; col++ {
				grid.SetCell(row, col, 0, 0)
			}
		}
		MarkDraw()
	}
}

// Sets cells with the given parameters, and advances y to the next. If
// `repeat` is present, the cell should be repeated `repeat` times (including
// the first time). This function will not check the end of the row. And
// currently only used by neovim.
func (manager *GridManager) SetCell(id, x int, y *int, char rune, attribId, repeat int) {
	grid, ok := manager.grids[id]
	if ok {
		for i := 0; i < common.Max(repeat, 1); i++ {
			grid.SetCell(x, *y, char, attribId)
			*y++
		}
	}
}

func (manager *GridManager) Update() {
	EndBenchmark := bench.BeginBenchmark()
	manager.HandleEvents()
	EndBenchmark("GridManager.Update")
}

// Rendering specific

func (manager *GridManager) Draw(force bool) {
	manager.SortGrids()
	for _, grid := range manager.sortedGrids {
		grid.Draw(force)
	}
}

func (manager *GridManager) Render() {
	for _, grid := range manager.sortedGrids {
		grid.Render()
	}
}
