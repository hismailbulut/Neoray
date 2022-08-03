package main

import (
	"reflect"
	"unicode"

	"github.com/hismailbulut/Neoray/pkg/common"
)

var (
	t_int  = reflect.TypeOf(int(0))
	t_uint = reflect.TypeOf(uint(0))
)

func (manager *GridManager) HandleEvents() {
	if !Editor.nvim.eventReceived.Get() {
		return
	}
	Editor.nvim.eventMutex.Lock()
	defer Editor.nvim.eventMutex.Unlock()
	// We must only take last cursor event at same redraw event batch, see issue #6
	gridCursorGotoLast := -1
	for i := len(Editor.nvim.eventStack) - 1; i >= 0; i-- {
		if Editor.nvim.eventStack[i][0] == "grid_cursor_goto" {
			gridCursorGotoLast = i
			break
		}
	}
	// Event stack is 3 dimensional array of interface, so it may contain an array also
	for i, update := range Editor.nvim.eventStack {
		switch update[0] {
		// Global events
		case "set_title":
			title := reflect.ValueOf(update[1]).Index(0).Elem().String()
			Editor.window.SetTitle(title)
		case "set_icon":
		case "mode_info_set":
			manager.mode_info_set(update[1:])
		case "option_set":
			manager.option_set(update[1:])
		case "mode_change":
			manager.mode_change(update[1:])
		case "mouse_on":
		case "mouse_off":
		case "busy_start":
			Editor.cursor.Hide()
		case "busy_stop":
			Editor.cursor.Show()
		case "suspend":
		case "update_menu":
		case "bell":
		case "visual_bell":
		case "flush":
			if Editor.state < EditorFirstFlush {
				SetEditorState(EditorFirstFlush)
			}
			MarkDraw()
		// Grid Events (line-based)
		case "grid_resize":
			manager.grid_resize(update[1:])
		case "default_colors_set":
			manager.default_colors_set(update[1:])
		case "hl_attr_define":
			manager.hl_attr_define(update[1:])
		case "hl_group_set":
		case "grid_line":
			manager.grid_line(update[1:])
		case "grid_clear":
			manager.grid_clear(update[1:])
		case "grid_destroy":
			manager.grid_destroy(update[1:])
		case "grid_cursor_goto":
			if gridCursorGotoLast == i {
				manager.grid_cursor_goto(update[1:])
			}
		case "grid_scroll":
			manager.grid_scroll(update[1:])
		// Multgrid specific events
		case "win_pos":
			manager.win_pos(update[1:])
		case "win_float_pos":
			manager.win_float_pos(update[1:])
		case "win_external_pos":
			manager.win_external_pos(update[1:])
		case "win_hide":
			manager.win_hide(update[1:])
		case "win_close":
			manager.win_close(update[1:])
		case "msg_set_pos":
			manager.msg_set_pos(update[1:])
		case "win_viewport":
			manager.win_viewport(update[1:])
		}
	}
	// clear update stack
	Editor.nvim.eventStack = Editor.nvim.eventStack[0:0]
	Editor.nvim.eventReceived.Set(false)
}

func refToInt(val reflect.Value) int {
	return int(val.Elem().Convert(t_int).Int())
}

func (manager *GridManager) option_set(args []interface{}) {
	options := &Editor.uiOptions
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

func (manager *GridManager) mode_info_set(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		Editor.cursor.mode.cursor_style_enabled = v.Index(0).Elem().Bool()
		Editor.cursor.mode.Clear()
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
			Editor.cursor.mode.Add(info)
		}
		MarkForceDraw()
	}
}

func (manager *GridManager) mode_change(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		Editor.cursor.mode.current_mode_name = v.Index(0).Elem().String()
		Editor.cursor.mode.current_mode = refToInt(v.Index(1))
	}
}

func (manager *GridManager) grid_resize(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		cols := refToInt(v.Index(1))
		rows := refToInt(v.Index(2))
		manager.ResizeGrid(grid, rows, cols)
	}
}

func (manager *GridManager) default_colors_set(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		fg := uint32(v.Index(0).Elem().Convert(t_uint).Uint())
		bg := uint32(v.Index(1).Elem().Convert(t_uint).Uint())
		sp := uint32(v.Index(2).Elem().Convert(t_uint).Uint())
		manager.foreground = common.ColorFromUint(fg)
		manager.background = common.ColorFromUint(bg)
		manager.special = common.ColorFromUint(sp)
		// NOTE: Unlike the corresponding |ui-grid-old| events, the screen is not
		// always cleared after sending this event. The UI must repaint the
		// screen with changed background color itself.
		MarkForceDraw()
	}
}

func (manager *GridManager) hl_attr_define(args []interface{}) {
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
				hl_attr.foreground = common.ColorFromUint(fg)
			case "background":
				bg := uint32(val.Convert(t_uint).Uint())
				hl_attr.background = common.ColorFromUint(bg)
			case "special":
				sp := uint32(val.Convert(t_uint).Uint())
				hl_attr.special = common.ColorFromUint(sp)
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
			case "underlineline":
				// hl_attr.underlineline = true
			case "undercurl":
				hl_attr.undercurl = true
			case "underdot":
				// hl_attr.underdot = true
			case "underdash":
				// hl_attr.underdash = true
			case "blend":
				// hl_attr.blend = int(val.Convert(t_uint).Uint())
			}
		}
		manager.attributes[id] = hl_attr
		MarkForceDraw()
	}
}

func (manager *GridManager) grid_line(args []interface{}) {
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
				// because otherwise we try to draw every space
				if unicode.IsSpace(char) {
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
			manager.SetCell(grid, row, &col, char, attribId, repeat)
		}
	}
}

func (manager *GridManager) grid_clear(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		manager.ClearGrid(grid)
	}
}

func (manager *GridManager) grid_destroy(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		manager.DestroyGrid(grid)
	}
}

func (manager *GridManager) grid_cursor_goto(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		row := refToInt(v.Index(1))
		col := refToInt(v.Index(2))
		Editor.cursor.SetPosition(grid, row, col, false)
	}
}

func (manager *GridManager) grid_scroll(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid_id := refToInt(v.Index(0))
		top := refToInt(v.Index(1))
		bot := refToInt(v.Index(2))
		left := refToInt(v.Index(3))
		right := refToInt(v.Index(4))
		rows := refToInt(v.Index(5))
		// cols := refToInt(v.Index(6))
		manager.ScrollGrid(grid_id, top, bot, rows, left, right)
	}
}

func (manager *GridManager) win_pos(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid_id := refToInt(v.Index(0))
		win := refToInt(v.Index(1))
		start_row := refToInt(v.Index(2))
		start_col := refToInt(v.Index(3))
		width := refToInt(v.Index(4))
		height := refToInt(v.Index(5))
		manager.SetGridPos(grid_id, win, start_row, start_col, height, width, GridTypeNormal)
	}
}

func (manager *GridManager) win_float_pos(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid_id := refToInt(v.Index(0))
		win := refToInt(v.Index(1))
		anchor := v.Index(2).Elem().String()
		anchor_grid_id := refToInt(v.Index(3))
		anchor_row := refToInt(v.Index(4))
		anchor_col := refToInt(v.Index(5))
		// focusable := v.Index(6).Elem().Bool()

		grid := manager.Grid(grid_id)
		anchor_grid := manager.Grid(anchor_grid_id)

		if grid != nil && anchor_grid != nil {
			row := anchor_grid.sRow + anchor_row
			col := anchor_grid.sCol + anchor_col

			// TODO: This needs to be revisited.
			switch anchor {
			case "NW":
			case "NE":
				col -= grid.cols
			case "SW":
				row -= grid.rows
			case "SE":
				col -= grid.cols
				row -= grid.rows
			}

			manager.SetGridPos(grid_id, win, row, col, grid.rows, grid.cols, GridTypeFloat)
		}
	}
}

func (manager *GridManager) win_external_pos(args []interface{}) {
	// NOTE: Currently not supported
	/*
		for _, arg := range args {
			// Not implemented
			v := reflect.ValueOf(arg)
			grid := refToInt(v.Index(0))
			win := refToInt(v.Index(1))
		}
	*/
}

func (manager *GridManager) win_hide(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		manager.HideGrid(grid)
	}
}

func (manager *GridManager) win_close(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid := refToInt(v.Index(0))
		manager.DestroyGrid(grid)
	}
}

func (manager *GridManager) msg_set_pos(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		grid_id := refToInt(v.Index(0))
		row := refToInt(v.Index(1))
		// scrolled := v.Index(2).Elem().Bool()
		// sep_char := v.Index(3).Elem().String()

		grid := manager.Grid(grid_id)
		default_grid := manager.Grid(1)

		if grid != nil && default_grid != nil {
			manager.SetGridPos(grid_id, grid.window, default_grid.sRow+row, default_grid.sCol, grid.rows, grid.cols, GridTypeMessage)
		}
	}
}

func (manager *GridManager) win_viewport(args []interface{}) {
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
