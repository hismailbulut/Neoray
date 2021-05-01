package main

import (
	"fmt"
	"reflect"
)

func HandleNvimRedrawEvents(proc *NvimProcess, w *Window) {
	// defer measure_execution_time("handle_nvim_updates")()

	proc.update_mutex.Lock()
	if len(proc.update_stack) <= 0 {
		proc.update_mutex.Unlock()
		return
	}
	updates_cpy := make([][]interface{}, len(proc.update_stack[0]))
	copy(updates_cpy, proc.update_stack[0])
	proc.update_stack = proc.update_stack[1:]
	proc.update_mutex.Unlock()

	for _, updates := range updates_cpy {
		switch updates[0] {
		// Global events
		case "set_title":
			title := reflect.ValueOf(updates[1]).Index(0).Elem().String()
			w.SetTitle(title)
			break
		case "set_icon":
			break
		case "mode_info_set":
			mode_info_set(w, updates[1:])
			break
		case "option_set":
			option_set(w, updates[1:])
			break
		case "mode_change":
			name := reflect.ValueOf(updates[1]).Index(0).Elem().String()
			w.mode.current_mode_name = name
			id := reflect.ValueOf(updates[1]).Index(1).Elem().Convert(reflect.TypeOf(int(0))).Int()
			w.mode.current_mode = int(id)
			break
		case "mouse_on":
			break
		case "mouse_off":
			break
		case "busy_start":
			break
		case "busy_stop":
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
			w.canvas.Draw(&w.grid, &w.mode, &w.cursor)
			break
		// Grid Events (line-based)
		case "grid_resize":
			grid_resize(w, updates[1:])
			break
		case "default_colors_set":
			default_colors_set(w, updates[1:])
			break
		case "hl_attr_define":
			hl_attr_define(w, updates[1:])
			break
		case "hl_group_set":
			break
		case "grid_line":
			grid_line(w, updates[1:])
			break
		case "grid_clear":
			w.grid.ClearCells()
			break
		case "grid_destroy":
			break
		case "grid_cursor_goto":
			grid_cursor_goto(w, updates[1:])
			break
		case "grid_scroll":
			grid_scroll(w, updates[1:])
			break
		}
	}
}

func option_set(w *Window, args []interface{}) {
	t := reflect.TypeOf(int(0))
	for _, opt := range args {
		valr := reflect.ValueOf(opt).Index(1).Elem()
		switch reflect.ValueOf(opt).Index(0).Elem().String() {
		case "arabicshape":
			w.options.arabicshape = valr.Bool()
			break
		case "ambiwidth":
			w.options.ambiwidth = valr.String()
			break
		case "emoji":
			w.options.emoji = valr.Bool()
			break
		case "guifont":
			w.options.guifont = valr.String()
			break
		case "guifontset":
			w.options.guifontset = valr.String()
			break
		case "guifontwide":
			w.options.guifontwide = valr.String()
			break
		case "linespace":
			w.options.linespace = int(valr.Convert(t).Int())
			break
		case "pumblend":
			w.options.pumblend = int(valr.Convert(t).Int())
			break
		case "showtabline":
			w.options.showtabline = int(valr.Convert(t).Int())
			break
		case "termguicolors":
			w.options.termguicolors = valr.Bool()
			break
		}
	}
	fmt.Println("Options Updated:", w.options)
}

func mode_info_set(w *Window, args []interface{}) {
	r := reflect.ValueOf(args).Index(0).Elem()
	w.mode.cursor_style_enabled = r.Index(0).Elem().Bool()
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
		w.mode.mode_infos[info.name] = info
	}
}

func grid_resize(w *Window, args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(int(0))
	width := r.Index(1).Elem().Convert(t).Int()
	height := r.Index(2).Elem().Convert(t).Int()
	w.grid.Resize(int(width), int(height))
}

func default_colors_set(w *Window, args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(uint(0))
	fg := r.Index(0).Elem().Convert(t).Uint()
	bg := r.Index(1).Elem().Convert(t).Uint()
	sp := r.Index(2).Elem().Convert(t).Uint()
	w.grid.default_fg = convert_rgb24_to_rgba(uint32(fg))
	w.grid.default_bg = convert_rgb24_to_rgba(uint32(bg))
	w.grid.default_sp = convert_rgb24_to_rgba(uint32(sp))
}

func hl_attr_define(w *Window, args []interface{}) {
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
		w.grid.attributes[id] = hl_attr
	}
}

func grid_line(w *Window, args []interface{}) {
	t := reflect.TypeOf(int(0))
	for _, arg := range args {
		r := reflect.ValueOf(arg)
		row := int(r.Index(1).Elem().Convert(t).Int())
		col_start := int(r.Index(2).Elem().Convert(t).Int())
		// cells is an array of arrays each with 1 to 3 elements
		cells := r.Index(3).Elem().Interface().([]interface{})
		hl_id := 0 // if hl_id is not present, we will use the last one
		for _, cell := range cells {
			// cell is a slice, can has 1 to 3 elements
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
			w.grid.SetCell(row, &col_start, char, hl_id, repeat)
		}
	}
}

func grid_cursor_goto(w *Window, args []interface{}) {
	t := reflect.TypeOf(int(0))
	r := reflect.ValueOf(args).Index(0).Elem()
	X := int(r.Index(1).Elem().Convert(t).Int())
	Y := int(r.Index(2).Elem().Convert(t).Int())
	w.cursor.SetPosition(X, Y)
}

func grid_scroll(w *Window, args []interface{}) {
	t := reflect.TypeOf(int(0))
	r := reflect.ValueOf(args).Index(0).Elem()
	top := r.Index(1).Elem().Convert(t).Int()
	bot := r.Index(2).Elem().Convert(t).Int()
	left := r.Index(3).Elem().Convert(t).Int()
	right := r.Index(4).Elem().Convert(t).Int()
	rows := r.Index(5).Elem().Convert(t).Int()
	//cols := r.Index(6).Elem().Convert(t).Int()
	w.grid.Scroll(int(top), int(bot), int(rows), int(left), int(right))
}
