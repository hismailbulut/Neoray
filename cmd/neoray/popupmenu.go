package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/sqweek/dialog"
)

var ButtonNames = []string{
	"Cut", "Copy", "Paste", "Select All", "Open File",
}

var MenuButtonEvents = map[string]func(){
	ButtonNames[0]: func() {
		EditorSingleton.nvim.Cut()
	},
	ButtonNames[1]: func() {
		EditorSingleton.nvim.Copy()
	},
	ButtonNames[2]: func() {
		EditorSingleton.nvim.WriteAtCursor(glfw.GetClipboardString())
	},
	ButtonNames[3]: func() {
		EditorSingleton.nvim.SelectAll()
	},
	ButtonNames[4]: func() {
		filename, err := dialog.File().Load()
		if err == nil && filename != "" && filename != " " {
			EditorSingleton.nvim.OpenFile(filename)
		}
	},
}

type PopupMenu struct {
	pos        IntVec2
	vertexData VertexDataStorage
	hidden     bool
	width      int
	height     int
	cells      [][]string
}

// Popup menu is mostly hardcoded.
// And this is because actually renderer is hardcoded as well.
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
	pmenu.cells = make([][]string, pmenu.height, pmenu.height)
	for i := range pmenu.cells {
		pmenu.cells[i] = make([]string, pmenu.width, pmenu.width)
	}
	pmenu.CreateCells()
	return pmenu
}

// Only call this function at initializing.
func (pmenu *PopupMenu) CreateCells() {
	// Loop through all cells and give them correct characters
	log_debug("Width:", pmenu.width, "Height:", pmenu.height)
	for x, row := range pmenu.cells {
		for y := range row {
			c := ' '
			if y != 0 && y != pmenu.width-1 {
				if y-1 < len(ButtonNames[x]) {
					c = rune(ButtonNames[x][y-1])
				}
			}
			pmenu.cells[x][y] = string(c)
		}
	}
}

func (pmenu *PopupMenu) CreateVertexData() {
	pmenu.vertexData = EditorSingleton.renderer.ReserveVertexData(pmenu.width * pmenu.height)
	for x, row := range pmenu.cells {
		for y, char := range row {
			cell_id := x*pmenu.width + y
			var atlasPos IntRect
			if char != "" && char != " " {
				atlasPos = EditorSingleton.renderer.GetCharacterAtlasPosition(char, false, false, false)
			}
			pmenu.vertexData.SetVertexTexPos(cell_id, atlasPos)
		}
	}
}

func (pmenu *PopupMenu) ShowAt(pos IntVec2) {
	pmenu.pos = pos
	fg := EditorSingleton.grid.default_bg
	bg := EditorSingleton.grid.default_fg
	for x, row := range pmenu.cells {
		for y := range row {
			cell_id := x*pmenu.width + y
			rect := IntRect{
				X: pos.X + y*EditorSingleton.cellWidth,
				Y: pos.Y + x*EditorSingleton.cellHeight,
				W: EditorSingleton.cellWidth,
				H: EditorSingleton.cellHeight,
			}
			pmenu.vertexData.SetVertexPos(cell_id, rect)
			pmenu.vertexData.SetVertexColor(cell_id, fg, bg)
		}
	}
	pmenu.hidden = false
	EditorSingleton.renderer.renderCall = true
}

func (pmenu *PopupMenu) Hide() {
	for x, row := range pmenu.cells {
		for y := range row {
			cell_id := x*pmenu.width + y
			pmenu.vertexData.SetVertexPos(cell_id, IntRect{})
		}
	}
	pmenu.hidden = true
	EditorSingleton.renderer.renderCall = true
}

func (pmenu *PopupMenu) GlobalRect() IntRect {
	return IntRect{
		X: pmenu.pos.X,
		Y: pmenu.pos.Y,
		W: pmenu.width * EditorSingleton.cellWidth,
		H: pmenu.height * EditorSingleton.cellHeight,
	}
}

// Returns true if given position intersects with menu,
// and if the position is on the button, returns button index.
func (pmenu *PopupMenu) Intersects(pos IntVec2) (bool, int) {
	menuRect := pmenu.GlobalRect()
	if pos.X >= menuRect.X && pos.Y >= menuRect.Y &&
		pos.X < menuRect.X+menuRect.W && pos.Y < menuRect.Y+menuRect.H {
		// Areas are intersecting. Now we need to find button under the cursor.
		// This is very simple. First we find the cell at the position.
		relativePos := IntVec2{
			X: pos.X - pmenu.pos.X,
			Y: pos.Y - pmenu.pos.Y,
		}
		row := relativePos.Y / EditorSingleton.cellHeight
		col := relativePos.X / EditorSingleton.cellWidth
		if col > 0 && col < pmenu.width-1 {
			return true, row
		}
		return true, -1
	}
	return false, -1
}

// Call this function when mouse moved.
func (pmenu *PopupMenu) MouseMove(pos IntVec2) {
	if !pmenu.hidden {
		ok, index := pmenu.Intersects(pos)
		if ok {
			pmenu.vertexData.SetAllVertexColors(
				EditorSingleton.grid.default_bg, EditorSingleton.grid.default_fg)
			if index != -1 {
				row := index
				// clear all buttons color
				if row < len(pmenu.cells) {
					for col := 1; col < pmenu.width-1; col++ {
						cell_id := row*pmenu.width + col
						pmenu.vertexData.SetVertexColor(
							cell_id, EditorSingleton.grid.default_fg, EditorSingleton.grid.default_bg)
					}
				}
				EditorSingleton.renderer.renderCall = true
			}
		} else {
			pmenu.Hide()
		}
	}
}

// Call this function when mouse clicked.
// If rightbutton is false (left button is pressed) and positions are
// intersecting, this function returns true. This means if this function
// returns true than you shouldn't send button event to neovim.
func (pmenu *PopupMenu) MouseClick(rightbutton bool, pos IntVec2) bool {
	if !rightbutton && !pmenu.hidden {
		// If positions intersects than call button click event, hide popup menu otherwise.
		ok, index := pmenu.Intersects(pos)
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
		// Open popup menu on this position
		pmenu.ShowAt(pos)
	}
	return false
}
