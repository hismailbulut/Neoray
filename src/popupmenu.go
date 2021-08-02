package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/sqweek/dialog"
)

var ButtonNames = []string{
	"Cut", "Copy", "Paste", "Select All", "Open File",
}

var MenuButtonEvents = map[string]func(){
	ButtonNames[0]: func() { //cut
		text := singleton.nvim.cutSelected()
		if text != "" {
			glfw.SetClipboardString(text)
		}
	},
	ButtonNames[1]: func() { //copy
		text := singleton.nvim.copySelected()
		if text != "" {
			glfw.SetClipboardString(text)
		}
	},
	ButtonNames[2]: func() { //paste
		singleton.nvim.paste(glfw.GetClipboardString())
	},
	ButtonNames[3]: func() { //select all
		singleton.nvim.selectAll()
	},
	ButtonNames[4]: func() { //open file
		filename, err := dialog.File().Load()
		if err == nil && filename != "" && filename != " " {
			singleton.nvim.openFile(filename)
		}
	},
}

type PopupMenu struct {
	pos        IntVec2
	vertexData VertexDataStorage
	hidden     bool
	width      int
	height     int
	cells      [][]rune
}

func CreatePopupMenu() PopupMenu {
	pmenu := PopupMenu{
		hidden: true,
	}
	// Find the longest text.
	longest := 0
	for _, name := range ButtonNames {
		if len(name) > longest {
			longest = len(name)
		}
	}
	// Create cells
	pmenu.width = longest + 2
	pmenu.height = len(ButtonNames)
	pmenu.cells = make([][]rune, pmenu.height, pmenu.height)
	for i := range pmenu.cells {
		pmenu.cells[i] = make([]rune, pmenu.width, pmenu.width)
	}
	pmenu.createCells()
	return pmenu
}

// Only call this function at initializing.
func (pmenu *PopupMenu) createCells() {
	// Loop through all cells and give them correct characters
	for x, row := range pmenu.cells {
		for y := range row {
			var c rune = 0
			if y != 0 && y != pmenu.width-1 {
				if y-1 < len(ButtonNames[x]) {
					c = rune(ButtonNames[x][y-1])
					if c == ' ' {
						c = 0
					}
				}
			}
			pmenu.cells[x][y] = c
		}
	}
}

func (pmenu *PopupMenu) createVertexData() {
	pmenu.vertexData = singleton.renderer.reserveVertexData(pmenu.width * pmenu.height)
	pmenu.updateChars()
}

func (pmenu *PopupMenu) updateChars() {
	for x, row := range pmenu.cells {
		for y, char := range row {
			cell_id := x*pmenu.width + y
			var atlasPos IntRect
			if char != 0 {
				atlasPos = singleton.renderer.getCharPos(
					char, false, false, false, false)
				// For multiwidth character.
				if atlasPos.W > singleton.cellWidth {
					atlasPos.W /= 2
				}
			}
			pmenu.vertexData.setCellTex1(cell_id, atlasPos)
		}
	}
}

func (pmenu *PopupMenu) ShowAt(pos IntVec2) {
	pmenu.pos = pos
	fg := singleton.gridManager.defaultBg
	bg := singleton.gridManager.defaultFg
	for x, row := range pmenu.cells {
		for y := range row {
			cell_id := x*pmenu.width + y
			rect := F32Rect{
				X: float32(pos.X + y*singleton.cellWidth),
				Y: float32(pos.Y + x*singleton.cellHeight),
				W: float32(singleton.cellWidth),
				H: float32(singleton.cellHeight),
			}
			pmenu.vertexData.setCellPos(cell_id, rect)
			pmenu.vertexData.setCellFg(cell_id, fg)
			pmenu.vertexData.setCellBg(cell_id, bg)
		}
	}
	pmenu.hidden = false
	singleton.render()
}

func (pmenu *PopupMenu) Hide() {
	for x, row := range pmenu.cells {
		for y := range row {
			cell_id := x*pmenu.width + y
			pmenu.vertexData.setCellPos(cell_id, F32Rect{})
		}
	}
	pmenu.hidden = true
	singleton.render()
}

func (pmenu *PopupMenu) globalRect() IntRect {
	return IntRect{
		X: pmenu.pos.X,
		Y: pmenu.pos.Y,
		W: pmenu.width * singleton.cellWidth,
		H: pmenu.height * singleton.cellHeight,
	}
}

// Returns true if given position intersects with menu,
// and if the position is on the button, returns button index.
func (pmenu *PopupMenu) intersects(pos IntVec2) (bool, int) {
	menuRect := pmenu.globalRect()
	if pos.inRect(menuRect) {
		// Areas are intersecting. Now we need to find button under the cursor.
		// This is very simple. First we find the cell at the position.
		relativePos := IntVec2{
			X: pos.X - pmenu.pos.X,
			Y: pos.Y - pmenu.pos.Y,
		}
		row := relativePos.Y / singleton.cellHeight
		col := relativePos.X / singleton.cellWidth
		if col > 0 && col < pmenu.width-1 {
			return true, row
		}
		return true, -1
	}
	return false, -1
}

// Call this function when mouse moved.
func (pmenu *PopupMenu) mouseMove(pos IntVec2) {
	if !pmenu.hidden {
		ok, index := pmenu.intersects(pos)
		if ok {
			// Fill all cells with default colors.
			for i := 0; i < pmenu.width*pmenu.height; i++ {
				pmenu.vertexData.setCellFg(i, singleton.gridManager.defaultBg)
				pmenu.vertexData.setCellBg(i, singleton.gridManager.defaultFg)
			}
			if index != -1 {
				row := index
				if row < len(pmenu.cells) {
					// Highlight this row.
					for col := 1; col < pmenu.width-1; col++ {
						cell_id := row*pmenu.width + col
						pmenu.vertexData.setCellFg(cell_id, singleton.gridManager.defaultFg)
						pmenu.vertexData.setCellBg(cell_id, singleton.gridManager.defaultBg)
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
func (pmenu *PopupMenu) mouseClick(rightbutton bool, pos IntVec2) bool {
	if !rightbutton && !pmenu.hidden {
		// If positions are intersecting then call button click event, hide popup menu otherwise.
		ok, index := pmenu.intersects(pos)
		if ok {
			if index != -1 {
				MenuButtonEvents[ButtonNames[index]]()
				pmenu.Hide()
			}
			return true
		} else {
			pmenu.Hide()
		}
	} else if rightbutton {
		// Open popup menu at this position
		pmenu.ShowAt(pos)
	}
	return false
}
