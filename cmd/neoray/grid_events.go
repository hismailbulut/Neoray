package main

import (
	"fmt"
	"unicode"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/neovim/go-client/nvim"
)

func (manager *GridManager) HandleEvents() {
	// We must only take last cursor event at same redraw event batch, see issue #6
	var lastGridCursorGoto []interface{}
	for len(Editor.nvim.eventChan) > 0 {
		event := <-Editor.nvim.eventChan
		switch event[0] {
		// Global events
		case "set_title":
			title := event[1].([]interface{})[0].(string)
			Editor.window.SetTitle(title)
		case "set_icon":
		case "mode_info_set":
			manager.mode_info_set(event[1:])
		case "option_set":
			manager.option_set(event[1:])
		case "mode_change":
			manager.mode_change(event[1:])
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
			manager.grid_resize(event[1:])
		case "default_colors_set":
			manager.default_colors_set(event[1:])
		case "hl_attr_define":
			manager.hl_attr_define(event[1:])
		case "hl_group_set":
		case "grid_line":
			manager.grid_line(event[1:])
		case "grid_clear":
			manager.grid_clear(event[1:])
		case "grid_destroy":
			manager.grid_destroy(event[1:])
		case "grid_cursor_goto":
			lastGridCursorGoto = event
		case "grid_scroll":
			manager.grid_scroll(event[1:])
		// Multgrid specific events
		case "win_pos":
			manager.win_pos(event[1:])
		case "win_float_pos":
			manager.win_float_pos(event[1:])
		case "win_external_pos":
			manager.win_external_pos(event[1:])
		case "win_hide":
			manager.win_hide(event[1:])
		case "win_close":
			manager.win_close(event[1:])
		case "msg_set_pos":
			manager.msg_set_pos(event[1:])
		case "win_viewport":
			manager.win_viewport(event[1:])
		}
	}
	if lastGridCursorGoto != nil {
		manager.grid_cursor_goto(lastGridCursorGoto[1:])
	}
}

func to_int(v interface{}) int {
	switch v := v.(type) {
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	default:
		panic(fmt.Errorf("to_int: unexpected type %T", v))
	}
}

func to_uint32(v interface{}) uint32 {
	switch v := v.(type) {
	case int64:
		return uint32(v)
	case uint64:
		return uint32(v)
	case float64:
		return uint32(v)
	default:
		panic(fmt.Errorf("to_uint32: unexpected type %T", v))
	}
}

func (manager *GridManager) option_set(args []interface{}) {
	options := &Editor.uiOptions
	for _, arg := range args {
		arg := arg.([]interface{})
		opt := arg[0].(string)
		val := arg[1]
		switch opt {
		case "arabicshape":
			options.arabicshape = val.(bool)
		case "ambiwidth":
			options.ambiwidth = val.(string)
		case "emoji":
			options.emoji = val.(bool)
		case "guifont":
			options.setGuiFont(val.(string))
		case "guifontset":
			options.guifontset = val.(string)
		case "guifontwide":
			options.guifontwide = val.(string)
		case "linespace":
			options.linespace = to_int(val)
		case "pumblend":
			options.pumblend = to_int(val)
		case "showtabline":
			options.showtabline = to_int(val)
		case "termguicolors":
			options.termguicolors = val.(bool)
		}
	}
}

func (manager *GridManager) mode_info_set(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		Editor.cursor.mode.cursor_style_enabled = arg[0].(bool)
		Editor.cursor.mode.Clear()
		for _, infos := range arg[1].([]interface{}) {
			infoMap := infos.(map[string]interface{})
			info := ModeInfo{}
			for k, v := range infoMap {
				switch k {
				case "cursor_shape":
					info.cursor_shape = v.(string)
				case "cell_percentage":
					info.cell_percentage = to_int(v)
				case "blinkwait":
					info.blinkwait = to_int(v)
				case "blinkon":
					info.blinkon = to_int(v)
				case "blinkoff":
					info.blinkoff = to_int(v)
				case "attr_id":
					info.attr_id = to_int(v)
				case "attr_id_lm":
					info.attr_id_lm = to_int(v)
				case "short_name":
					info.short_name = v.(string)
				case "name":
					info.name = v.(string)
				}
			}
			Editor.cursor.mode.Add(info)
		}
	}
}

func (manager *GridManager) mode_change(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		Editor.cursor.mode.current_mode_name = arg[0].(string)
		Editor.cursor.mode.current_mode = to_int(arg[1])
	}
}

func (manager *GridManager) grid_resize(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		cols := to_int(arg[1])
		rows := to_int(arg[2])
		manager.ResizeGrid(grid_id, rows, cols)
	}
}

func (manager *GridManager) default_colors_set(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		fg := to_uint32(arg[0])
		bg := to_uint32(arg[1])
		sp := to_uint32(arg[2])
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
		arg := arg.([]interface{})
		// args is an array with first element is
		// attribute hl_id and second is a map which
		// contains attribute keys
		hl_id := to_int(arg[0])
		attribs := arg[1].(map[string]interface{})
		hl_attr := HighlightAttribute{}
		// iterate over map and set attributes
		for k, v := range attribs {
			switch k {
			case "foreground":
				fg := to_uint32(v)
				hl_attr.foreground = common.ColorFromUint(fg)
			case "background":
				bg := to_uint32(v)
				hl_attr.background = common.ColorFromUint(bg)
			case "special":
				sp := to_uint32(v)
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
		manager.attributes[hl_id] = hl_attr
		MarkForceDraw()
	}
}

func (manager *GridManager) grid_line(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		row := to_int(arg[1])
		col := to_int(arg[2])
		// cells is an array of arrays each with 1 to 3 elements
		cells := arg[3].([]interface{})
		hl_id := 0 // if hl_id is not present, we will use the last one
		for _, cell := range cells {
			// cell is a slice, may have 1 to 3 elements
			cell := cell.([]interface{})
			// first one is character
			var char rune
			str := cell[0].(string)
			if len(str) > 0 {
				char = []rune(str)[0]
				// If this is a space, we set it to zero
				// because otherwise we try to draw every space
				if unicode.IsSpace(char) {
					char = 0
				}
			}
			// second one is highlight attribute id -optional
			if len(cell) >= 2 {
				hl_id = to_int(cell[1])
			}
			// third one is repeat count -optional
			repeat := 0
			if len(cell) == 3 {
				repeat = to_int(cell[2])
			}
			manager.SetCell(grid_id, row, &col, char, hl_id, repeat)
		}
	}
}

func (manager *GridManager) grid_clear(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		manager.ClearGrid(grid_id)
	}
}

func (manager *GridManager) grid_destroy(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		manager.DestroyGrid(grid_id)
	}
}

func (manager *GridManager) grid_cursor_goto(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		row := to_int(arg[1])
		col := to_int(arg[2])
		Editor.cursor.SetPosition(grid_id, row, col)
	}
}

func (manager *GridManager) grid_scroll(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		top := to_int(arg[1])
		bot := to_int(arg[2])
		left := to_int(arg[3])
		right := to_int(arg[4])
		rows := to_int(arg[5])
		// cols := to_int(v[6])
		manager.ScrollGrid(grid_id, top, bot, rows, left, right)
	}
}

func (manager *GridManager) win_pos(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		win := arg[1].(nvim.Window)
		start_row := to_int(arg[2])
		start_col := to_int(arg[3])
		width := to_int(arg[4])
		height := to_int(arg[5])
		manager.SetGridPos(grid_id, win, start_row, start_col, height, width, GridTypeNormal)
	}
}

func (manager *GridManager) win_float_pos(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		win := arg[1].(nvim.Window)
		anchor := arg[2].(string)
		anchor_grid_id := to_int(arg[3])
		anchor_row := to_int(arg[4])
		anchor_col := to_int(arg[5])
		// focusable := v[6].(bool)

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
			arg := arg.([]interface{})
			grid := to_int(arg[0])
			win := arg[1].(nvim.Window)
		}
	*/
}

func (manager *GridManager) win_hide(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		manager.HideGrid(grid_id)
	}
}

func (manager *GridManager) win_close(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		manager.DestroyGrid(grid_id)
	}
}

func (manager *GridManager) msg_set_pos(args []interface{}) {
	for _, arg := range args {
		arg := arg.([]interface{})
		grid_id := to_int(arg[0])
		row := to_int(arg[1])
		// scrolled := arg[2].(bool)
		// sep_char := arg[3].(string)

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
			arg := arg.([]interface{})
			grid_id := to_int(arg[0])
			win := arg[1].(nvim.Window)
			topline := to_int(arg[2])
			botline := to_int(arg[3])
			curline := to_int(arg[4])
			curcol := to_int(arg[5])
		}
	*/
}
