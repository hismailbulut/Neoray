package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/fontkit"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/sqweek/dialog"
)

type ContextButton struct {
	name string
	fn   func()
}

// You can add more buttons here.
var ContextMenuButtons = []ContextButton{
	{
		name: "Cut",
		fn: func() {
			text := Editor.nvim.Cut()
			if text != "" {
				glfw.SetClipboardString(text)
			}
		},
	},
	{
		name: "Copy",
		fn: func() {
			text := Editor.nvim.Copy()
			if text != "" {
				glfw.SetClipboardString(text)
			}
		},
	},
	{
		name: "Paste",
		fn: func() {
			Editor.nvim.Paste(glfw.GetClipboardString())
		},
	},
	{
		name: "Select All",
		fn: func() {
			Editor.nvim.SelectAll()
		},
	},
	{
		name: "Open File",
		fn: func() {
			filename, err := dialog.File().Load()
			if err == nil && filename != "" {
				Editor.nvim.EditFile(filename)
			}
			Editor.window.Raise()
		},
	},
}

type ContextMenu struct {
	pos        common.Vector2[int]
	hidden     bool
	rows, cols int
	cells      [][]rune
	hlRow      int // Highlighted row index, -1 if none
	renderer   *GridRenderer
}

func NewContextMenu() *ContextMenu {
	menu := new(ContextMenu)
	menu.hidden = true
	menu.createCells()
	menu.hlRow = -1
	var err error
	menu.renderer, err = NewGridRenderer(Editor.window, menu.rows, menu.cols, nil, DEFAULT_FONT_SIZE, menu.pos)
	if err != nil {
		logger.Log(logger.ERROR, "Failed to create context menu renderer")
	}
	return menu
}

func (menu *ContextMenu) createCells() {
	// Find the longest text.
	longest := 0
	for _, btn := range ContextMenuButtons {
		if len(btn.name) > longest {
			longest = len(btn.name)
		}
	}
	// Create cells
	menu.cols = longest + 2
	menu.rows = len(ContextMenuButtons)
	menu.cells = make([][]rune, menu.rows)
	for i := range menu.cells {
		menu.cells[i] = make([]rune, menu.cols)
	}
	// Resize renderer
	if menu.renderer != nil {
		menu.renderer.Resize(menu.rows, menu.cols)
	}
	// Loop through all cells and give them correct characters
	for row := 0; row < menu.rows; row++ {
		for col := 1; col < menu.cols-1; col++ {
			var c rune = 0
			if col-1 < len(ContextMenuButtons[row].name) {
				c = rune(ContextMenuButtons[row].name[col-1])
				if c == ' ' {
					c = 0
				}
			}
			menu.cells[row][col] = c
		}
	}
}

func (menu *ContextMenu) SetFontKit(kit *fontkit.FontKit) {
	menu.renderer.SetFontKit(kit)
	MarkForceDraw()
}

func (menu *ContextMenu) SetFontSize(size float64) {
	menu.renderer.SetFontSize(size, Editor.window.DPI())
	MarkForceDraw()
}

func (menu *ContextMenu) AddFontSize(v float64) {
	size := menu.renderer.FontSize() + v
	menu.SetFontSize(size)
	MarkForceDraw()
}

func (menu *ContextMenu) IsVisible() bool {
	return !menu.hidden
}

func (menu *ContextMenu) Draw() {
	if menu.hidden {
		return
	}
	EndBenchmark := bench.Begin()
	for row := 0; row < menu.rows; row++ {
		for col := 0; col < menu.cols; col++ {
			char := menu.cells[row][col]
			if menu.hlRow == row && col > 0 && col < menu.cols-1 {
				// This is the highlighted cell
				menu.renderer.DrawCell(row, col, char, HighlightAttribute{
					foreground: Editor.gridManager.foreground,
					background: Editor.gridManager.background,
					bold:       true,
				})
			} else {
				// Normally we swap menu colors
				menu.renderer.DrawCell(row, col, char, HighlightAttribute{
					foreground: Editor.gridManager.background,
					background: Editor.gridManager.foreground,
					bold:       true,
				})
			}
		}
	}
	EndBenchmark("ContextMenu.Draw")
}

func (menu *ContextMenu) Render() {
	if menu.hidden {
		return
	}
	menu.renderer.Render()
}

func (menu *ContextMenu) AddButton(button ContextButton) {
	ContextMenuButtons = append(ContextMenuButtons, button)
	menu.createCells()
}

func (menu *ContextMenu) ShowAt(pos common.Vector2[int]) {
	menu.hidden = false
	menu.pos = pos
	menu.renderer.SetPos(pos)
	MarkDraw()
}

func (menu *ContextMenu) Hide() {
	if !menu.hidden {
		menu.hidden = true
		MarkRender()
	}
}

func (menu *ContextMenu) Dimensions() common.Rectangle[int] {
	cellSize := menu.renderer.CellSize()
	return common.Rectangle[int]{
		X: menu.pos.X,
		Y: menu.pos.Y,
		W: menu.cols * cellSize.Width(),
		H: menu.rows * cellSize.Height(),
	}
}

// Returns true if given position IsIntersecting with menu,
// and if the position is on the button, returns button index.
func (menu *ContextMenu) IsIntersecting(pos common.Vector2[int]) (bool, int) {
	menuRect := menu.Dimensions()
	if pos.IsInRect(menuRect) {
		// Areas are intersecting. Now we need to find button under the cursor.
		// This is very simple. First we find the cell at the position.
		relativePos := common.Vector2[int]{
			X: pos.X - menu.pos.X,
			Y: pos.Y - menu.pos.Y,
		}
		cellSize := menu.renderer.CellSize()
		row := relativePos.Y / cellSize.Height()
		col := relativePos.X / cellSize.Width()
		if col > 0 && col < menu.cols-1 {
			return true, row
		}
		return true, -1
	}
	return false, -1
}

// Call this function when mouse moved.
func (menu *ContextMenu) MouseMove(pos common.Vector2[int]) {
	if !menu.hidden {
		ok, index := menu.IsIntersecting(pos)
		if ok {
			// The index is not -1 means cursor is on top of a button. And
			// index is the index of the button and also row of the popup menu.
			if index != -1 {
				if index < len(menu.cells) {
					// Highlight this row.
					if menu.hlRow != index {
						menu.hlRow = index
						MarkDraw()
					}
				}
			} else {
				if menu.hlRow != -1 {
					menu.hlRow = -1
					MarkDraw()
				}
			}
		} else {
			// If this uncommented, the context menu will be hidden
			// when cursor goes out from on top of it.
			// pmenu.Hide()
			// Clear highlight
			if menu.hlRow != -1 {
				menu.hlRow = -1
				MarkDraw()
			}
		}
	}
}

// Call this function when mouse clicked.
// If rightbutton is false (left button is pressed) and positions are
// intersecting, this function returns true. This means if this function
// returns true than you shouldn't send button event to neovim.
func (menu *ContextMenu) MouseClick(rightbutton bool, pos common.Vector2[int]) bool {
	if !rightbutton && !menu.hidden {
		// If positions are intersecting then call button click event, hide popup menu otherwise.
		ok, index := menu.IsIntersecting(pos)
		if ok {
			if index != -1 {
				ContextMenuButtons[index].fn()
				menu.Hide()
			}
		} else {
			menu.Hide()
		}
		return true
	} else if rightbutton {
		// Open popup menu at this position
		menu.ShowAt(pos)
		return true
	}
	return false
}

func (menu *ContextMenu) Destroy() {
	menu.renderer.Destroy()
	logger.Log(logger.DEBUG, "Context menu destroyed")
}
