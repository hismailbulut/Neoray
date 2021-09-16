package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/sqweek/dialog"
)

type ContextButton struct {
	name string
	fn   func()
}

// You can add more buttons here.
var ContextMenuButtons = []ContextButton{
	{name: "Cut",
		fn: func() {
			text := singleton.nvim.cutSelected()
			if text != "" {
				glfw.SetClipboardString(text)
			}
		}},
	{name: "Copy",
		fn: func() {
			text := singleton.nvim.copySelected()
			if text != "" {
				glfw.SetClipboardString(text)
			}
		}},
	{name: "Paste",
		fn: func() {
			singleton.nvim.paste(glfw.GetClipboardString())
		}},
	{name: "Select All",
		fn: func() {
			singleton.nvim.selectAll()
		}},
	{name: "Open File",
		fn: func() {
			filename, err := dialog.File().Load()
			if err == nil && filename != "" {
				singleton.nvim.openFile(filename)
			}
			singleton.window.raise()
		}},
}

type ContextMenu struct {
	pos        IntVec2
	vertexData VertexDataStorage
	hidden     bool
	width      int
	height     int
	cells      [][]rune
}

func CreateContextMenu() ContextMenu {
	cMenu := ContextMenu{
		hidden: true,
	}
	cMenu.createCells()
	return cMenu
}

// Only call this function at initializing.
func (cMenu *ContextMenu) createCells() {
	// Find the longest text.
	longest := 0
	for _, btn := range ContextMenuButtons {
		if len(btn.name) > longest {
			longest = len(btn.name)
		}
	}
	// Create cells
	cMenu.width = longest + 2
	cMenu.height = len(ContextMenuButtons)
	cMenu.cells = make([][]rune, cMenu.height, cMenu.height)
	for i := range cMenu.cells {
		cMenu.cells[i] = make([]rune, cMenu.width, cMenu.width)
	}
	// Loop through all cells and give them correct characters
	for x, row := range cMenu.cells {
		for y := range row {
			var c rune = 0
			if y != 0 && y != cMenu.width-1 {
				if y-1 < len(ContextMenuButtons[x].name) {
					c = rune(ContextMenuButtons[x].name[y-1])
					if c == ' ' {
						c = 0
					}
				}
			}
			cMenu.cells[x][y] = c
		}
	}
}

func (cMenu *ContextMenu) createVertexData() {
	cMenu.vertexData = singleton.renderer.reserveVertexData(cMenu.width * cMenu.height)
	cMenu.updateChars()
}

func (cMenu *ContextMenu) updateChars() {
	for x, row := range cMenu.cells {
		for y, char := range row {
			cell_id := x*cMenu.width + y
			var atlasPos IntRect
			if char != 0 {
				atlasPos = singleton.renderer.getCharPos(
					char, false, false, false, false)
				// For multiwidth character.
				if atlasPos.W > singleton.cellWidth {
					atlasPos.W /= 2
				}
			}
			cMenu.vertexData.setCellTex1(cell_id, atlasPos)
		}
	}
}

func (cMenu *ContextMenu) AddButton(button ContextButton) {
	ContextMenuButtons = append(ContextMenuButtons, button)
	singleton.contextMenu.createCells()
	if singleton.mainLoopRunning {
		singleton.renderer.createVertexData()
		singleton.contextMenu.updateChars()
		singleton.fullDraw()
	}
}

func (cMenu *ContextMenu) ShowAt(pos IntVec2) {
	cMenu.pos = pos
	fg := singleton.gridManager.defaultBg
	bg := singleton.gridManager.defaultFg
	for x, row := range cMenu.cells {
		for y := range row {
			cell_id := x*cMenu.width + y
			rect := F32Rect{
				X: float32(pos.X + y*singleton.cellWidth),
				Y: float32(pos.Y + x*singleton.cellHeight),
				W: float32(singleton.cellWidth),
				H: float32(singleton.cellHeight),
			}
			cMenu.vertexData.setCellPos(cell_id, rect)
			cMenu.vertexData.setCellFg(cell_id, fg)
			cMenu.vertexData.setCellBg(cell_id, bg)
		}
	}
	cMenu.hidden = false
	singleton.render()
}

func (cMenu *ContextMenu) Hide() {
	for x, row := range cMenu.cells {
		for y := range row {
			cell_id := x*cMenu.width + y
			cMenu.vertexData.setCellPos(cell_id, F32Rect{})
		}
	}
	cMenu.hidden = true
	singleton.render()
}

func (cMenu *ContextMenu) globalRect() IntRect {
	return IntRect{
		X: cMenu.pos.X,
		Y: cMenu.pos.Y,
		W: cMenu.width * singleton.cellWidth,
		H: cMenu.height * singleton.cellHeight,
	}
}

// Returns true if given position intersects with menu,
// and if the position is on the button, returns button index.
func (cMenu *ContextMenu) intersects(pos IntVec2) (bool, int) {
	menuRect := cMenu.globalRect()
	if pos.inRect(menuRect) {
		// Areas are intersecting. Now we need to find button under the cursor.
		// This is very simple. First we find the cell at the position.
		relativePos := IntVec2{
			X: pos.X - cMenu.pos.X,
			Y: pos.Y - cMenu.pos.Y,
		}
		row := relativePos.Y / singleton.cellHeight
		col := relativePos.X / singleton.cellWidth
		if col > 0 && col < cMenu.width-1 {
			return true, row
		}
		return true, -1
	}
	return false, -1
}

// Call this function when mouse moved.
func (cMenu *ContextMenu) mouseMove(pos IntVec2) {
	if !cMenu.hidden {
		// Fill all cells with default colors.
		for i := 0; i < cMenu.width*cMenu.height; i++ {
			cMenu.vertexData.setCellFg(i, singleton.gridManager.defaultBg)
			cMenu.vertexData.setCellBg(i, singleton.gridManager.defaultFg)
		}
		ok, index := cMenu.intersects(pos)
		if ok {
			// The index is not -1 means cursor is on top of a button. And
			// index is the index of the button and also row of the popup menu.
			if index != -1 {
				if index < len(cMenu.cells) {
					// Highlight this row.
					for col := 1; col < cMenu.width-1; col++ {
						cell_id := index*cMenu.width + col
						cMenu.vertexData.setCellFg(cell_id, singleton.gridManager.defaultFg)
						cMenu.vertexData.setCellBg(cell_id, singleton.gridManager.defaultBg)
					}
				}
				singleton.render()
			}
		} else {
			// If this uncommented, the context menu will be hidden
			// when cursor goes out from on top of it.
			// pmenu.Hide()
		}
	}
}

// Call this function when mouse clicked.
// If rightbutton is false (left button is pressed) and positions are
// intersecting, this function returns true. This means if this function
// returns true than you shouldn't send button event to neovim.
func (cMenu *ContextMenu) mouseClick(rightbutton bool, pos IntVec2) bool {
	if !rightbutton && !cMenu.hidden {
		// If positions are intersecting then call button click event, hide popup menu otherwise.
		ok, index := cMenu.intersects(pos)
		if ok {
			if index != -1 {
				ContextMenuButtons[index].fn()
				cMenu.Hide()
			}
		} else {
			cMenu.Hide()
		}
		return true
	} else if rightbutton {
		// Open popup menu at this position
		cMenu.ShowAt(pos)
		return true
	}
	return false
}
