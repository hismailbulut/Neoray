package main

import (
	"reflect"
)

var (
	t_int  = reflect.TypeOf(int(0))
	t_uint = reflect.TypeOf(uint(0))
)

func handleRedrawEvents() {
	if singleton.nvim.eventReceived.Get() {
		singleton.nvim.eventMutex.Lock()
		defer singleton.nvim.eventMutex.Unlock()
		for _, updates := range singleton.nvim.eventStack {
			for _, update := range updates {
				switch update[0] {
				// Global events
				case "set_title":
					title := reflect.ValueOf(update[1]).Index(0).Elem().String()
					singleton.window.setTitle(title)
				case "set_icon":
					break
				case "mode_info_set":
					mode_info_set(update[1:])
				case "option_set":
					option_set(update[1:])
				case "mode_change":
					mode_change(update[1:])
				case "mouse_on":
					break
				case "mouse_off":
					break
				case "busy_start":
					singleton.cursor.Hide()
				case "busy_stop":
					singleton.cursor.Show()
				case "suspend":
					break
				case "update_menu":
					break
				case "bell":
					break
				case "visual_bell":
					break
				case "flush":
					singleton.draw()
				// Grid Events (line-based)
				case "grid_resize":
					grid_resize(update[1:])
				case "default_colors_set":
					default_colors_set(update[1:])
				case "hl_attr_define":
					hl_attr_define(update[1:])
				case "hl_group_set":
					break
				case "grid_line":
					grid_line(update[1:])
				case "grid_clear":
					grid_clear(update[1:])
				case "grid_destroy":
					grid_destroy(update[1:])
				case "grid_cursor_goto":
					grid_cursor_goto(update[1:])
				case "grid_scroll":
					grid_scroll(update[1:])
				// Multgrid specific events
				case "win_pos":
					win_pos(update[1:])
				case "win_float_pos":
					win_float_pos(update[1:])
				case "win_external_pos":
					win_external_pos(update[1:])
				case "win_hide":
					win_hide(update[1:])
				case "win_close":
					win_close(update[1:])
				case "msg_set_pos":
					msg_set_pos(update[1:])
				case "win_viewport":
					win_viewport(update[1:])
				}
			}
		}
		// clear update stack
		singleton.nvim.eventStack = singleton.nvim.eventStack[0:0]
		singleton.nvim.eventReceived.Set(false)
	}
}

func refToInt(val reflect.Value) int {
	return int(val.Elem().Convert(t_int).Int())
}

func option_set(args []interface{}) {
	options := &singleton.uiOptions
	for _, arg := range args {
		val := reflect.ValueOf(arg).Index(1).Elem()
		switch reflect.ValueOf(arg).Index(0).Elem().String() {
		case "arabicshape":
			options.arabicshape = val.Bool()
		case "ambiwidth":
			options.ambiwidth = val.String()
		case "emoji":
			options.emoji = val.Bool()
		case "guifont":
			options.setGuiFont(val.String())
		case "guifontset":
			options.guifontset = val.String()
		case "guifontwide":
			options.guifontwide = val.String()
		case "linespace":
			options.linespace = int(val.Convert(t_int).Int())
		case "pumblend":
			options.pumblend = int(val.Convert(t_int).Int())
		case "showtabline":
			options.showtabline = int(val.Convert(t_int).Int())
		case "termguicolors":
			options.termguicolors = val.Bool()
		}
	}
}

func mode_info_set(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		singleton.mode.cursor_style_enabled = v.Index(0).Elem().Bool()
		singleton.mode.Clear()
		for _, infos := range v.Index(1).Interface().([]interface{}) {
			mapIter := reflect.ValueOf(infos).MapRange()
			info := ModeInfo{}
			for mapIter.Next() {
				val := mapIter.Value().Elem()
				switch mapIter.Key().String() {
				case "cursor_shape":
					info.cursor_shape = val.String()
				case "cell_percentage":
					info.cell_percentage = int(val.Convert(t_int).Int())
				case "blinkwait":
					info.blinkwait = int(val.Convert(t_int).Int())
				case "blinkon":
					info.blinkon = int(val.Convert(t_int).Int())
				case "blinkoff":
					info.blinkoff = int(val.Convert(t_int).Int())
				case "attr_id":
					info.attr_id = int(val.Convert(t_int).Int())
				case "attr_id_lm":
					info.attr_id_lm = int(val.Convert(t_int).Int())
				case "short_name":
					info.short_name = val.String()
				case "name":
					info.name = val.String()
				}
			}
			singleton.mode.Add(info)
		}
		singleton.cursor.needsDraw = true
	}
}

func mode_change(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		singleton.mode.current_mode_name = v.Index(0).Elem().String()
		singleton.mode.current_mode = refToInt(v.Index(1))
	}
}

func grid_resize(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		cols := refToInt(v.Index(1))
		rows := refToInt(v.Index(2))

		singleton.gridManager.resize(grid, rows, cols)

		// Grid 1 is the default grid for entire screen.
		if grid == 1 {
			singleton.renderer.resize(rows, cols)
		}
	}
}

func default_colors_set(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		fg := v.Index(0).Elem().Convert(t_uint).Uint()
		bg := v.Index(1).Elem().Convert(t_uint).Uint()
		sp := v.Index(2).Elem().Convert(t_uint).Uint()
		singleton.gridManager.defaultFg = unpackColor(uint32(fg))
		singleton.gridManager.defaultBg = unpackColor(uint32(bg))
		singleton.gridManager.defaultSp = unpackColor(uint32(sp))
		// NOTE: Unlike the corresponding |ui-grid-old| events, the screen is not
		// always cleared after sending this event. The UI must repaint the
		// screen with changed background color itself.
		singleton.fullDraw()
	}
}

func hl_attr_define(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		// args is an array with first element is
		// attribute id and second is a map which
		// contains attribute keys
		id := refToInt(v.Index(0))
		mapIter := v.Index(1).Elem().MapRange()
		hl_attr := HighlightAttribute{}
		// iterate over map and set attributes
		for mapIter.Next() {
			val := mapIter.Value().Elem()
			switch mapIter.Key().String() {
			case "foreground":
				fg := uint32(val.Convert(t_uint).Uint())
				hl_attr.foreground = unpackColor(fg)
			case "background":
				bg := uint32(val.Convert(t_uint).Uint())
				hl_attr.background = unpackColor(bg)
			case "special":
				sp := uint32(val.Convert(t_uint).Uint())
				hl_attr.special = unpackColor(sp)
			// All boolean keys default to false,
			// and will only be sent when they are true.
			case "reverse":
				hl_attr.reverse = true
			case "italic":
				hl_attr.italic = true
			case "bold":
				hl_attr.bold = true
			case "strikethrough":
				hl_attr.strikethrough = true
			case "underline":
				hl_attr.underline = true
			case "undercurl":
				hl_attr.undercurl = true
			case "blend":
				hl_attr.blend = int(val.Convert(t_uint).Uint())
			}
		}
		singleton.gridManager.attributes[id] = hl_attr
		singleton.fullDraw()
	}
}

func grid_line(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		row := refToInt(v.Index(1))
		col := refToInt(v.Index(2))
		// cells is an array of arrays each with 1 to 3 elements
		cells := v.Index(3).Elem().Interface().([]interface{})
		attribId := 0 // if hl_id is not present, we will use the last one
		for _, cell := range cells {
			// cell is a slice, may have 1 to 3 elements
			cellv := reflect.ValueOf(cell)
			// first one is character
			var char rune
			str := cellv.Index(0).Elem().String()
			if len(str) > 0 {
				char = []rune(str)[0]
				// If this is a space, we set it to zero
				// because otherwise we draw every space
				if char == ' ' {
					char = 0
				}
			}
			// second one is highlight attribute id -optional
			if cellv.Len() >= 2 {
				attribId = refToInt(cellv.Index(1))
			}
			// third one is repeat count -optional
			repeat := 0
			if cellv.Len() == 3 {
				repeat = refToInt(cellv.Index(2))
			}
			singleton.gridManager.setCell(grid, row, &col, char, attribId, repeat)
		}
	}
	singleton.draw()
}

func grid_clear(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		singleton.gridManager.clear(grid)
	}
}

func grid_destroy(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		singleton.gridManager.destroy(grid)
	}
}

func grid_cursor_goto(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		x := refToInt(v.Index(1))
		y := refToInt(v.Index(2))
		singleton.cursor.setPosition(grid, x, y, false)
	}
}

func grid_scroll(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		top := refToInt(v.Index(1))
		bot := refToInt(v.Index(2))
		left := refToInt(v.Index(3))
		right := refToInt(v.Index(4))
		rows := refToInt(v.Index(5))
		// cols := refToInt(v.Index(6))

		singleton.gridManager.grids[grid].scroll(top, bot, rows, left, right)
	}
}

func win_pos(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		win := refToInt(v.Index(1))
		start_row := refToInt(v.Index(2))
		start_col := refToInt(v.Index(3))
		width := refToInt(v.Index(4))
		height := refToInt(v.Index(5))

		singleton.gridManager.grids[grid].setPos(win, start_row, start_col, height, width, GridTypeNormal)
	}
}

func win_float_pos(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		win := refToInt(v.Index(1))
		anchor := v.Index(2).Elem().String()
		anchor_grid := refToInt(v.Index(3))
		anchor_row := refToInt(v.Index(4))
		anchor_col := refToInt(v.Index(5))
		// focusable := v.Index(6).Elem().Bool()

		currentGrid := singleton.gridManager.grids[grid]
		anchorGrid := singleton.gridManager.grids[anchor_grid]

		row := anchorGrid.sRow + anchor_row
		col := anchorGrid.sCol + anchor_col

		// TODO: This needs to be revisited.
		switch anchor {
		case "NW":
		case "NE":
			col -= currentGrid.cols
		case "SW":
			row -= currentGrid.rows
		case "SE":
			col -= currentGrid.cols
			row -= currentGrid.rows
		}

		currentGrid.setPos(win, row, col, currentGrid.rows, currentGrid.cols, GridTypeFloat)
	}
}

func win_external_pos(args []interface{}) {
	// NOTE: Creating an external window needs hard work. Because of this we
	// are not support external windows for now.
	/*
		for _, arg := range args {
			// Not implemented
			v := reflect.ValueOf(arg)
			grid := refToInt(v.Index(0))
			win := refToInt(v.Index(1))
		}
	*/
}

func win_hide(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		singleton.gridManager.hide(grid)
	}
}

func win_close(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		singleton.gridManager.destroy(grid)
	}
}

func msg_set_pos(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		row := refToInt(v.Index(1))
		// scrolled := v.Index(2).Elem().Bool()
		// sep_char := v.Index(3).Elem().String()

		currentGrid := singleton.gridManager.grids[grid]
		defaultGrid := singleton.gridManager.grids[1]

		currentGrid.setPos(currentGrid.window, defaultGrid.sRow+row, defaultGrid.sCol,
			currentGrid.rows, currentGrid.cols, GridTypeMessage)
	}
}

func win_viewport(args []interface{}) {
	/*
		for _, arg := range args {
			v := reflect.ValueOf(arg)
			grid := refToInt(v.Index(0))
			win := refToInt(v.Index(1))
			topline := refToInt(v.Index(2))
			botline := refToInt(v.Index(3))
			curline := refToInt(v.Index(4))
			curcol := refToInt(v.Index(5))
		}
	*/
}
