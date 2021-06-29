package main

import (
	"reflect"
)

var (
	t_int  = reflect.TypeOf(int(0))
	t_uint = reflect.TypeOf(uint(0))
)

func HandleNvimRedrawEvents() {
	defer measure_execution_time()()

	EditorSingleton.nvim.update_mutex.Lock()
	defer EditorSingleton.nvim.update_mutex.Unlock()

	if len(EditorSingleton.nvim.update_stack) <= 0 {
		return
	}

	for _, updates := range EditorSingleton.nvim.update_stack {
		for _, update := range updates {
			switch update[0] {
			// Global events
			case "set_title":
				title := reflect.ValueOf(update[1]).Index(0).Elem().String()
				EditorSingleton.window.setTitle(title)
				break
			case "set_icon":
				break
			case "mode_info_set":
				mode_info_set(update[1:])
				break
			case "option_set":
				option_set(update[1:])
				break
			case "mode_change":
				mode_change(update[1:])
				break
			case "mouse_on":
				break
			case "mouse_off":
				break
			case "busy_start":
				EditorSingleton.cursor.Hide()
				break
			case "busy_stop":
				EditorSingleton.cursor.Show()
				break
			case "suspend":
				break
			case "update_menu":
				break
			case "bell":
				log_debug(update...)
				break
			case "visual_bell":
				log_debug(update...)
				break
			case "flush":
				EditorSingleton.draw()
				break
			// Grid Events (line-based)
			case "grid_resize":
				grid_resize(update[1:])
				break
			case "default_colors_set":
				default_colors_set(update[1:])
				break
			case "hl_attr_define":
				hl_attr_define(update[1:])
				break
			case "hl_group_set":
				break
			case "grid_line":
				grid_line(update[1:])
				break
			case "grid_clear":
				EditorSingleton.grid.clearCells()
				break
			case "grid_destroy":
				break
			case "grid_cursor_goto":
				grid_cursor_goto(update[1:])
				break
			case "grid_scroll":
				grid_scroll(update[1:])
				break
			default:
				log_debug("Unhandled redraw event:", update)
				break
			}
		}
	}
	// clear update stack
	EditorSingleton.nvim.update_stack = nil
}

func option_set(args []interface{}) {
	options := &EditorSingleton.options
	for _, arg := range args {
		val := reflect.ValueOf(arg).Index(1).Elem()
		switch reflect.ValueOf(arg).Index(0).Elem().String() {
		case "arabicshape":
			options.arabicshape = val.Bool()
			break
		case "ambiwidth":
			options.ambiwidth = val.String()
			break
		case "emoji":
			options.emoji = val.Bool()
			break
		case "guifont":
			options.SetGuiFont(val.String())
			break
		case "guifontset":
			options.guifontset = val.String()
			break
		case "guifontwide":
			options.guifontwide = val.String()
			break
		case "linespace":
			options.linespace = int(val.Convert(t_int).Int())
			break
		case "pumblend":
			options.pumblend = int(val.Convert(t_int).Int())
			break
		case "showtabline":
			options.showtabline = int(val.Convert(t_int).Int())
			break
		case "termguicolors":
			options.termguicolors = val.Bool()
			break
		}
	}
}

func mode_info_set(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		EditorSingleton.mode.cursor_style_enabled = v.Index(0).Elem().Bool()
		EditorSingleton.mode.Clear()
		for _, infos := range v.Index(1).Interface().([]interface{}) {
			mapIter := reflect.ValueOf(infos).MapRange()
			info := ModeInfo{}
			for mapIter.Next() {
				val := mapIter.Value().Elem()
				switch mapIter.Key().String() {
				case "cursor_shape":
					info.cursor_shape = val.String()
					break
				case "cell_percentage":
					info.cell_percentage = int(val.Convert(t_int).Int())
					break
				case "blinkwait":
					info.blinkwait = int(val.Convert(t_int).Int())
					break
				case "blinkon":
					info.blinkon = int(val.Convert(t_int).Int())
					break
				case "blinkoff":
					info.blinkoff = int(val.Convert(t_int).Int())
					break
				case "attr_id":
					info.attr_id = int(val.Convert(t_int).Int())
					break
				case "attr_id_lm":
					info.attr_id_lm = int(val.Convert(t_int).Int())
					break
				case "short_name":
					info.short_name = val.String()
					break
				case "name":
					info.name = val.String()
					break
				}
			}
			EditorSingleton.mode.Add(info)
		}
		EditorSingleton.cursor.needsDraw = true
	}
}

func mode_change(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		name := v.Index(0).Elem().String()
		EditorSingleton.mode.current_mode_name = name
		id := v.Index(1).Elem().Convert(t_int).Int()
		EditorSingleton.mode.current_mode = int(id)
	}
}

func grid_resize(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		_ = int(v.Index(0).Elem().Convert(t_int).Int())
		cols := int(v.Index(1).Elem().Convert(t_int).Int())
		rows := int(v.Index(2).Elem().Convert(t_int).Int())

		EditorSingleton.rowCount = rows
		EditorSingleton.columnCount = cols
		EditorSingleton.cellCount = rows * cols

		EditorSingleton.grid.resize(rows, cols)
		EditorSingleton.renderer.resize(rows, cols)
		EditorSingleton.waitingResize = false
	}
}

func default_colors_set(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		fg := v.Index(0).Elem().Convert(t_uint).Uint()
		bg := v.Index(1).Elem().Convert(t_uint).Uint()
		sp := v.Index(2).Elem().Convert(t_uint).Uint()
		EditorSingleton.grid.default_fg = unpackColor(uint32(fg))
		EditorSingleton.grid.default_bg = unpackColor(uint32(bg))
		EditorSingleton.grid.default_sp = unpackColor(uint32(sp))
		// NOTE: Unlike the corresponding |ui-grid-old| events, the screen is not
		// always cleared after sending this event. The UI must repaint the
		// screen with changed background color itself.
		// Maybe not clear all cells but it's working. And without
		// it it's also working.
		EditorSingleton.grid.clearCells()
	}
}

func hl_attr_define(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		// args is an array with first element is
		// attribute id and second is a map which
		// contains attribute keys
		id := int(v.Index(0).Elem().Convert(t_uint).Uint())
		assert(id != 0, "hl id is zero")
		// if id == 0 {
		//     // `id` 0 will always be used for the default highlight with colors
		//     continue
		// }
		mapIter := v.Index(1).Elem().MapRange()
		hl_attr := HighlightAttribute{}
		// iterate over map and set attributes
		for mapIter.Next() {
			val := mapIter.Value().Elem()
			switch mapIter.Key().String() {
			case "foreground":
				fg := uint32(val.Convert(t_uint).Uint())
				hl_attr.foreground = unpackColor(fg)
				break
			case "background":
				bg := uint32(val.Convert(t_uint).Uint())
				hl_attr.background = unpackColor(bg)
				break
			case "special":
				sp := uint32(val.Convert(t_uint).Uint())
				hl_attr.special = unpackColor(sp)
				break
			// All boolean keys default to false,
			// and will only be sent when they are true.
			case "reverse":
				hl_attr.reverse = true
				break
			case "italic":
				hl_attr.italic = true
				break
			case "bold":
				hl_attr.bold = true
				break
			case "strikethrough":
				hl_attr.strikethrough = true
				break
			case "underline":
				hl_attr.underline = true
				break
			case "undercurl":
				hl_attr.undercurl = true
				break
			case "blend":
				hl_attr.blend = int(val.Convert(t_uint).Uint())
				break
			}
		}
		EditorSingleton.grid.attributes[id] = hl_attr
		EditorSingleton.grid.makeAllCellsChanged()
	}
}

func grid_line(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		_ = int(v.Index(1).Elem().Convert(t_int).Int())
		row := int(v.Index(1).Elem().Convert(t_int).Int())
		col_start := int(v.Index(2).Elem().Convert(t_int).Int())
		// cells is an array of arrays each with 1 to 3 elements
		cells := v.Index(3).Elem().Interface().([]interface{})
		hl_id := 0 // if hl_id is not present, we will use the last one
		for _, cell := range cells {
			// cell is a slice, may have 1 to 3 elements
			// first one is character
			// second one is highlight attribute id -optional
			// third one is repeat count -optional
			cellv := reflect.ValueOf(cell)
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
			repeat := 0
			if cellv.Len() >= 2 {
				hl_id = int(cellv.Index(1).Elem().Convert(t_int).Int())
			}
			if cellv.Len() == 3 {
				repeat = int(cellv.Index(2).Elem().Convert(t_int).Int())
			}
			EditorSingleton.grid.setCells(row, &col_start, char, hl_id, repeat)
		}
	}
}

func grid_cursor_goto(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		_ = int(v.Index(0).Elem().Convert(t_int).Int())
		X := int(v.Index(1).Elem().Convert(t_int).Int())
		Y := int(v.Index(2).Elem().Convert(t_int).Int())
		EditorSingleton.cursor.SetPosition(X, Y, false)
	}
}

func grid_scroll(args []interface{}) {
	for _, arg := range args {
		v := reflect.ValueOf(arg)
		_ = v.Index(0).Elem().Convert(t_int).Int()
		top := v.Index(1).Elem().Convert(t_int).Int()
		bot := v.Index(2).Elem().Convert(t_int).Int()
		left := v.Index(3).Elem().Convert(t_int).Int()
		right := v.Index(4).Elem().Convert(t_int).Int()
		rows := v.Index(5).Elem().Convert(t_int).Int()
		//cols := r.Index(6).Elem().Convert(t).Int()
		EditorSingleton.grid.scroll(int(top), int(bot), int(rows), int(left), int(right))
	}
}
