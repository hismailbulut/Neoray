package main

import (
	rl "github.com/chunqian/go-raylib/raylib"
)

type Cell struct {
	char      string
	attrib_id int
}

type HighlightAttributes struct {
	foreground rl.Color
	background rl.Color
	special    rl.Color
	reverse    bool
	italic     bool
	bold       bool
	// strikethrough
	underline bool
	undercurl bool
	// blend
}

type GridTable struct {
	cells        [][]Cell
	width        int
	height       int
	default_fg   rl.Color
	default_bg   rl.Color
	default_sp   rl.Color
	attributes   []HighlightAttributes
	changed_rows map[int]bool
}

func CreateGridTable() GridTable {
	table := GridTable{
		attributes: make([]HighlightAttributes, 256),
	}
	return table
}

func (table *GridTable) Resize(width int, height int) {
	table.width = width
	table.height = height
	// row count is height, column count is width
	table.cells = make([][]Cell, height) // rows
	for i := range table.cells {
		table.cells[i] = make([]Cell, width) // columns
	}
	// initialize and set changed rows
	table.changed_rows = make(map[int]bool, len(table.cells))
	for i := 0; i < len(table.cells); i++ {
		table.changed_rows[i] = true
	}
}

func (table *GridTable) ClearCells() {
	for i, row := range table.cells {
		for _, cell := range row {
			cell.char = ""
			cell.attrib_id = 0
		}
		table.changed_rows[i] = true
	}
}

func (table *GridTable) SetCell(x int, y *int, char string, hl_id int, repeat int) {
	// If `repeat` is present, the cell should be
	// repeated `repeat` times (including the first time)
	if repeat == 0 {
		table.cells[x][*y].char = char
		table.cells[x][*y].attrib_id = hl_id
		*y++
	} else {
		for i := 0; i < repeat; i++ {
			table.cells[x][*y].char = char
			table.cells[x][*y].attrib_id = hl_id
			*y++
		}
	}
	table.changed_rows[x] = true
}

func (table *GridTable) SetHlAttribute(id int, fg, bg, sp uint32, reverse, italic, bold, underline, undercurl bool) {
	attrib := HighlightAttributes{
		foreground: convert_rgb24_to_rgba(fg),
		background: convert_rgb24_to_rgba(bg),
		special:    convert_rgb24_to_rgba(sp),
		reverse:    reverse,
		italic:     italic,
		bold:       bold,
		underline:  underline,
		undercurl:  undercurl,
	}
	attr_id := id - 1
	if int(attr_id) == len(table.attributes) {
		// new attribute, append
		table.attributes = append(table.attributes, attrib)
	} else if int(attr_id) < len(table.attributes) {
		// set attribute
		table.attributes[attr_id] = attrib
	}
}

func (table *GridTable) Scroll(top, bot, rows, left, right int) {
	if rows > 0 { // Scroll down, move up
		for y := top + rows; y < bot; y++ { // row
			copy(table.cells[y-rows][left:right], table.cells[y][left:right])
			table.changed_rows[y-rows] = true
		}
	} else { // Scroll up, move down
		// rows is negative
		rows = -rows
		for y := (bot - rows) - 1; y >= top; y-- { // row
			copy(table.cells[y+rows][left:right], table.cells[y][left:right])
			table.changed_rows[y+rows] = true
		}
	}
}
