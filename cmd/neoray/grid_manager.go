package main

import (
	"sort"

	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/neovim/go-client/nvim"
)

type GridManager struct {
	editor      *Editor
	grids       map[int]*Grid
	sortedGrids []*Grid
	// These are used for creating new grids
	totalGridsCreated int     // total number of grids created (including deleted ones)
	fontSize          float64 // last globally set font size
	// style information
	attributes map[int]HighlightAttribute
	foreground common.Color // Default foreground color
	background common.Color // Default background color
	special    common.Color // Default special color
}

func NewGridManager(editor *Editor) *GridManager {
	grid := &GridManager{
		editor:     editor,
		grids:      make(map[int]*Grid),
		attributes: make(map[int]HighlightAttribute),
	}
	return grid
}

// Font related

// This is used only when the window scale has been changed in runtime (DPI)
func (manager *GridManager) ResetFontSize() {
	for _, grid := range manager.grids {
		grid.SetFontSize(grid.renderer.FontSize(), manager.editor.window.DPI())
		manager.CheckGridSize(grid.id)
	}
}

func (manager *GridManager) SetGridFontSize(id int, fontSize float64) {
	if id == 1 {
		for _, grid := range manager.grids {
			grid.SetFontSize(fontSize, manager.editor.window.DPI())
			manager.CheckGridSize(grid.id)
		}
		manager.fontSize = fontSize
	} else {
		grid := manager.Grid(id)
		if grid != nil {
			grid.SetFontSize(fontSize, manager.editor.window.DPI())
			manager.CheckGridSize(id)
		}
	}
	manager.editor.MarkForceDraw()
}

func (manager *GridManager) IncrementGridFontSize(id int, v float64) {
	if id == 1 {
		manager.fontSize += v
		manager.SetGridFontSize(id, manager.fontSize)
	} else {
		grid := manager.Grid(id)
		if grid != nil {
			manager.SetGridFontSize(id, grid.renderer.FontSize()+v)
		}
	}
}

func (manager *GridManager) SetBoxDrawing(useBoxDrawing, useBlockDrawing bool) {
	for _, grid := range manager.grids {
		grid.SetBoxDrawing(useBoxDrawing, useBlockDrawing)
	}
	manager.editor.MarkForceDraw()
}

func (manager *GridManager) CheckDefaultGridSize() {
	manager.CheckGridSize(1)
}

func (manager *GridManager) CheckGridSize(id int) {
	grid := manager.Grid(id)
	if grid == nil {
		return
	}
	var area common.Vector2[int]
	if id == 1 {
		area = manager.editor.window.Size()
	} else {
		area.X = grid.rect.Dx()
		area.Y = grid.rect.Dy()
	}
	cell := grid.CellSize()
	rows := area.Height() / cell.Height()
	cols := area.Width() / cell.Width()
	if rows == grid.rows && cols == grid.cols {
		return
	}
	if id == 1 {
		manager.editor.nvim.TryResizeUI(rows, cols)
	} else {
		manager.editor.nvim.TryResizeUIGrid(grid.id, rows, cols)
	}
}

// Sorts grids according to rendering order and returns it.
// You can access the sorted array via gridManager.sortedGrids
// and don't call this function directly.
func (manager *GridManager) SortGrids() {
	bench.Begin()()
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

func (manager *GridManager) CellAttribute(id int) HighlightAttribute {
	if id == 0 {
		// Default attribute
		background := manager.editor.gridManager.background
		background.A = manager.editor.options.transparency
		return HighlightAttribute{
			foreground: manager.editor.gridManager.foreground,
			background: background,
			special:    manager.editor.gridManager.special,
		}
	} else {
		attrib, ok := manager.editor.gridManager.attributes[id]
		if !ok {
			logger.Errorf("Attribute id %d not found!", id)
			return attrib
		}
		// Zero alpha means color is not set and we use default color
		if attrib.foreground.A <= 0 {
			attrib.foreground = manager.editor.gridManager.foreground
		}
		if attrib.background.A <= 0 {
			attrib.background = manager.editor.gridManager.background
			// Default backgrounds are transparent
			attrib.background.A = manager.editor.options.transparency
		}
		if attrib.special.A <= 0 {
			attrib.special = manager.editor.gridManager.special
		}
		// Reverse foreground an background colors if reverse attribute set
		if attrib.reverse {
			attrib.foreground, attrib.background = attrib.background, attrib.foreground
			attrib.reverse = false
		}
		return attrib
	}
}

// Returns grid id and cell position at the given global position.
// The returned values are grid id, cell row, cell column
// Returned grid is always 1 if multigrid is off
func (manager *GridManager) CellAt(pos common.Vector2[int]) (int, int, int) {
	id, row, col := -1, -1, -1
	// The input_mouse api call wants 0 for grid when multigrid is not enabled
	if !manager.editor.parsedArgs.multiGrid {
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
		attrib, ok := manager.editor.gridManager.attributes[cell.attribID]
		if !ok {
			attrib = HighlightAttribute{}
		}
		vertex := grid.renderer.CellVertexData(row, col)
		font := manager.editor.fontManager.MustSuitableFont(false, false, -1)
		pos, _ := grid.renderer.atlas.GetCharPos(font, cell.char, attrib.bold, attrib.italic, attrib.underline, attrib.strikethrough)
		format := `Cell Info (Grid: %d Row: %d Col: %d)
	%v
	%v
	Position in atlas: %v
	Attrib:
		Fg:   %v
		Bg:   %v
		Sp:   %v
		Bold:          %v
		Italic:        %v
		Underline:     %v
		Strikethrough: %v
		Undercurl:     %v
	Vertex:
		Pos:  %v Area: %f
		Tex1: %v Area: %f
		Tex2: %v Area: %f
		Fg:   %v
		Bg:   %v
		Sp:   %v`
		logger.Debugf(format,
			gridID, row, col,
			cell,
			grid,
			pos,
			attrib.foreground,
			attrib.background,
			attrib.special,
			attrib.bold,
			attrib.italic,
			attrib.underline,
			attrib.strikethrough,
			attrib.undercurl,
			vertex.Pos, vertex.Pos.Area(),
			vertex.Tex1, vertex.Tex1.Area(),
			vertex.Tex2, vertex.Tex2.Area(),
			vertex.Fg,
			vertex.Bg,
			vertex.Sp,
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

func (manager *GridManager) SetGridPos(id int, win nvim.Window, sRow, sCol, rows, cols int, typ GridType) {
	grid, ok := manager.grids[id]
	if ok {
		position := manager.GridPosition(sRow, sCol)
		grid.SetPos(win, sRow, sCol, rows, cols, typ, position)
	}
}

// This function returns nil if there is no grid with this id exists
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
		grid, err = NewGrid(manager.editor, id, manager.totalGridsCreated, rows, cols, manager.fontSize, common.Vec2(0, 0))
		if err != nil {
			logger.Fatal("Grid creation failed:", err)
		}
		manager.grids[id] = grid
	}
	manager.editor.MarkForceDraw()
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
		manager.editor.MarkForceDraw()
	}
}

func (manager *GridManager) DestroyGrid(id int) {
	grid, ok := manager.grids[id]
	if ok {
		delete(manager.grids, id)
		grid.Destroy()
		manager.editor.MarkForceDraw()
	}
}

func (manager *GridManager) Destroy() {
	for k := range manager.grids {
		manager.DestroyGrid(k)
	}
	logger.Debug("Grid manager destroyed")
}

func (manager *GridManager) ClearGrid(id int) {
	grid, ok := manager.grids[id]
	if ok {
		for row := 0; row < grid.rows; row++ {
			for col := 0; col < grid.cols; col++ {
				grid.SetCell(row, col, 0, 0)
			}
		}
		manager.editor.MarkDraw()
	}
}

// Sets the cells at the (x, [y, y+repeat]) to the char and attr, advances y to the end
func (manager *GridManager) SetCell(id, x int, y *int, char rune, attribId, repeat int) {
	grid, ok := manager.grids[id]
	if !ok {
		return
	}
	for i := 0; i < repeat; i++ {
		if grid.IsInBounds(x, *y) {
			grid.SetCell(x, *y, char, attribId)
			*y++
		}
	}
}

func (manager *GridManager) Update() {
	EndBenchmark := bench.Begin()
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
