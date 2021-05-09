package main

import (
	"reflect"
)

// this function is directly called from nvim process in other goroutine
// we are handling some thread safe events here
// NOTE: Reserved
func redraw_events_prehandler(updates *[][]interface{}) {
	for _, update := range *updates {
		switch update[0] {
		default:
		}
	}
}

func HandleNvimRedrawEvents(editor *Editor) {
	// defer measure_execution_time("handle_nvim_updates")()

	editor.nvim.update_mutex.Lock()
	if len(editor.nvim.update_stack) <= 0 {
		editor.nvim.update_mutex.Unlock()
		return
	}
	updates_cpy := make([][]interface{}, len(editor.nvim.update_stack[0]))
	copy(updates_cpy, editor.nvim.update_stack[0])
	editor.nvim.update_stack = editor.nvim.update_stack[1:]
	editor.nvim.update_mutex.Unlock()

	for _, updates := range updates_cpy {
		switch updates[0] {
		// Global events
		case "set_title":
			title := reflect.ValueOf(updates[1]).Index(0).Elem().String()
			editor.window.SetTitle(title)
			break
		case "set_icon":
			break
		case "mode_info_set":
			mode_info_set(&editor.mode, updates[1:])
			break
		case "option_set":
			option_set(&editor.options, updates[1:])
			break
		case "mode_change":
			name := reflect.ValueOf(updates[1]).Index(0).Elem().String()
			editor.mode.current_mode_name = name
			id := reflect.ValueOf(updates[1]).Index(1).Elem().Convert(reflect.TypeOf(int(0))).Int()
			editor.mode.current_mode = int(id)
			break
		case "mouse_on":
			break
		case "mouse_off":
			break
		case "busy_start":
			log_debug_msg("Busy started.", updates)
			break
		case "busy_stop":
			log_debug_msg("Busy stopped.", updates)
			break
		case "suspend":
			break
		case "update_menu":
			break
		case "bell":
			break
		case "visual_bell":
			break
		case "flush":
			editor.renderer.Draw(&editor.grid, &editor.mode, &editor.cursor)
			break
		// Grid Events (line-based)
		case "grid_resize":
			grid_resize(&editor.grid, updates[1:])
			break
		case "default_colors_set":
			default_colors_set(&editor.grid, updates[1:])
			break
		case "hl_attr_define":
			hl_attr_define(&editor.grid, updates[1:])
			break
		case "hl_group_set":
			break
		case "grid_line":
			grid_line(&editor.grid, updates[1:])
			break
		case "grid_clear":
			editor.grid.ClearCells()
			break
		case "grid_destroy":
			break
		case "grid_cursor_goto":
			grid_cursor_goto(&editor.cursor, updates[1:])
			break
		case "grid_scroll":
			grid_scroll(&editor.grid, updates[1:])
			break
		}
	}
}

func option_set(options *UIOptions, args []interface{}) {
	t := reflect.TypeOf(int(0))
	for _, opt := range args {
		valr := reflect.ValueOf(opt).Index(1).Elem()
		switch reflect.ValueOf(opt).Index(0).Elem().String() {
		case "arabicshape":
			options.arabicshape = valr.Bool()
			break
		case "ambiwidth":
			options.ambiwidth = valr.String()
			break
		case "emoji":
			options.emoji = valr.Bool()
			break
		case "guifont":
			options.guifont = valr.String()
			break
		case "guifontset":
			options.guifontset = valr.String()
			break
		case "guifontwide":
			options.guifontwide = valr.String()
			break
		case "linespace":
			options.linespace = int(valr.Convert(t).Int())
			break
		case "pumblend":
			options.pumblend = int(valr.Convert(t).Int())
			break
		case "showtabline":
			options.showtabline = int(valr.Convert(t).Int())
			break
		case "termguicolors":
			options.termguicolors = valr.Bool()
			break
		}
	}
}

func mode_info_set(mode *Mode, args []interface{}) {
	r := reflect.ValueOf(args).Index(0).Elem()
	mode.cursor_style_enabled = r.Index(0).Elem().Bool()
	t := reflect.TypeOf(int(0))
	for _, infos := range r.Index(1).Interface().([]interface{}) {
		mapIter := reflect.ValueOf(infos).MapRange()
		info := ModeInfo{}
		for mapIter.Next() {
			switch mapIter.Key().String() {
			case "cursor_shape":
				info.cursor_shape = mapIter.Value().Elem().String()
				break
			case "cell_percentage":
				info.cell_percentage = int(mapIter.Value().Elem().Convert(t).Int())
				break
			case "blinkwait":
				info.blinkwait = int(mapIter.Value().Elem().Convert(t).Int())
				break
			case "blinkon":
				info.blinkon = int(mapIter.Value().Elem().Convert(t).Int())
				break
			case "blinkoff":
				info.blinkoff = int(mapIter.Value().Elem().Convert(t).Int())
				break
			case "attr_id":
				info.attr_id = int(mapIter.Value().Elem().Convert(t).Int())
				break
			case "attr_id_lm":
				info.attr_id_lm = int(mapIter.Value().Elem().Convert(t).Int())
				break
			case "short_name":
				info.short_name = mapIter.Value().Elem().String()
				break
			case "name":
				info.name = mapIter.Value().Elem().String()
				break
			}
		}
		mode.mode_infos[info.name] = info
	}
}

func grid_resize(grid *Grid, args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(int(0))
	width := r.Index(1).Elem().Convert(t).Int()
	height := r.Index(2).Elem().Convert(t).Int()
	grid.Resize(int(width), int(height))
}

func default_colors_set(grid *Grid, args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(uint(0))
	fg := r.Index(0).Elem().Convert(t).Uint()
	bg := r.Index(1).Elem().Convert(t).Uint()
	sp := r.Index(2).Elem().Convert(t).Uint()
	grid.default_fg = convert_rgb24_to_rgba(uint32(fg))
	grid.default_bg = convert_rgb24_to_rgba(uint32(bg))
	grid.default_sp = convert_rgb24_to_rgba(uint32(sp))
}

func hl_attr_define(grid *Grid, args []interface{}) {
	// there may be more than one attribute
	t := reflect.TypeOf(uint(0))
	for _, arg := range args {
		// args is an array with first element is
		// attribute id and second is a map which
		// contains attribute keys
		id := int(reflect.ValueOf(arg).Index(0).Elem().Convert(t).Uint())
		if id == 0 {
			// `id` 0 will always be used for the default highlight with colors
			continue
		}
		mapIter := reflect.ValueOf(arg).Index(1).Elem().MapRange()
		hl_attr := HighlightAttributes{}
		// iterate over map and set attributes
		for mapIter.Next() {
			switch mapIter.Key().String() {
			case "foreground":
				fg := uint32(mapIter.Value().Elem().Convert(t).Uint())
				hl_attr.foreground = convert_rgb24_to_rgba(fg)
				break
			case "background":
				bg := uint32(mapIter.Value().Elem().Convert(t).Uint())
				hl_attr.background = convert_rgb24_to_rgba(bg)
				break
			case "special":
				sp := uint32(mapIter.Value().Elem().Convert(t).Uint())
				hl_attr.special = convert_rgb24_to_rgba(sp)
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
				hl_attr.blend = int(mapIter.Value().Elem().Convert(t).Uint())
				break
			}
		}
		grid.attributes[id] = hl_attr
	}
}

func grid_line(grid *Grid, args []interface{}) {
	t := reflect.TypeOf(int(0))
	for _, arg := range args {
		r := reflect.ValueOf(arg)
		row := int(r.Index(1).Elem().Convert(t).Int())
		col_start := int(r.Index(2).Elem().Convert(t).Int())
		// cells is an array of arrays each with 1 to 3 elements
		cells := r.Index(3).Elem().Interface().([]interface{})
		hl_id := 0 // if hl_id is not present, we will use the last one
		for _, cell := range cells {
			// cell is a slice, may have 1 to 3 elements
			// first one is character
			// second one is highlight attribute id -optional
			// third one is repeat count -optional
			cell_slice := cell.([]interface{})
			char := cell_slice[0].(string)
			repeat := 0
			if len(cell_slice) >= 2 {
				hl_id = int(reflect.ValueOf(cell_slice).Index(1).Elem().Convert(t).Int())
			}
			if len(cell_slice) == 3 {
				repeat = int(reflect.ValueOf(cell_slice).Index(2).Elem().Convert(t).Int())
			}
			grid.SetCell(row, &col_start, char, hl_id, repeat)
		}
	}
}

func grid_cursor_goto(cursor *Cursor, args []interface{}) {
	t := reflect.TypeOf(int(0))
	r := reflect.ValueOf(args).Index(0).Elem()
	X := int(r.Index(1).Elem().Convert(t).Int())
	Y := int(r.Index(2).Elem().Convert(t).Int())
	cursor.SetPosition(X, Y)
}

func grid_scroll(grid *Grid, args []interface{}) {
	t := reflect.TypeOf(int(0))
	r := reflect.ValueOf(args).Index(0).Elem()
	top := r.Index(1).Elem().Convert(t).Int()
	bot := r.Index(2).Elem().Convert(t).Int()
	left := r.Index(3).Elem().Convert(t).Int()
	right := r.Index(4).Elem().Convert(t).Int()
	rows := r.Index(5).Elem().Convert(t).Int()
	//cols := r.Index(6).Elem().Convert(t).Int()
	grid.Scroll(int(top), int(bot), int(rows), int(left), int(right))
}
