package main

import (
	"fmt"
	"reflect"
)

func handle_nvim_updates(proc *NvimProcess, w *Window) {
	// defer measure_execution_time("handle_nvim_updates")()
	if len(proc.update_stack) <= 0 {
		return
	}
	for _, updates := range proc.update_stack[0] {
		switch updates[0] {
		// Global events
		case "set_title":
			title := reflect.ValueOf(updates[1]).Index(0).Elem().String()
			w.SetTitle(title)
			break
		case "set_icon":
			fmt.Println("Update: Set Icon:", updates[1:])
			break
		case "mode_info_set":
			fmt.Println("Update: Mode Info Set:", updates[1:])
			break
		case "option_set":
			fmt.Println("Update: Option Set:", updates[1:])
			break
		case "mode_change":
			fmt.Println("Update: Mode Change:", updates[1:])
			break
		case "mouse_on":
			fmt.Println("Update: Mouse On:", updates[1:])
			break
		case "mouse_off":
			fmt.Println("Update: Mouse Off:", updates[1:])
			break
		case "busy_start":
			fmt.Println("Update: Busy Start:", updates[1:])
			break
		case "busy_stop":
			fmt.Println("Update: Busy Stop:", updates[1:])
			break
		case "suspend":
			fmt.Println("Update: Suspend:", updates[1:])
			break
		case "update_menu":
			fmt.Println("Update: Update Menu:", updates[1:])
			break
		case "bell":
			fmt.Println("Update: Bell:", updates[1:])
			break
		case "visual_bell":
			fmt.Println("Update: Visual Bell:", updates[1:])
			break
		case "flush":
			w.canvas.Draw(&w.table)
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
			w.table.ClearCells()
			break
		case "grid_destroy":
			fmt.Println("Update: Grid Destroy:", updates[1:])
			break
		case "grid_cursor_goto":
			grid_cursor_goto(w, updates[1:])
			break
		case "grid_scroll":
			grid_scroll(w, updates[1:])
			break
		}
	}
	proc.update_stack = proc.update_stack[1:]
}

func grid_resize(w *Window, args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(int(0))
	width := r.Index(1).Elem().Convert(t).Int()
	height := r.Index(2).Elem().Convert(t).Int()
	w.table.Resize(int(width), int(height))
}

func default_colors_set(w *Window, args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(uint(0))
	fg := r.Index(0).Elem().Convert(t).Uint()
	bg := r.Index(1).Elem().Convert(t).Uint()
	sp := r.Index(2).Elem().Convert(t).Uint()
	w.table.default_fg = convert_rgb24_to_rgba(uint32(fg))
	w.table.default_bg = convert_rgb24_to_rgba(uint32(bg))
	w.table.default_sp = convert_rgb24_to_rgba(uint32(sp))
}

func hl_attr_define(w *Window, args []interface{}) {
	// there may be more than one attribute
	t := reflect.TypeOf(uint(0))
	for _, arg := range args {
		// args is an array with first element is
		// attribute id and second is a map which
		// contains attribute keys
		id := reflect.ValueOf(arg).Index(0).Elem().Convert(t).Uint()
		if id == 0 {
			// `id` 0 will always be used for the default highlight with colors
			continue
		}
		mapIter := reflect.ValueOf(arg).Index(1).Elem().MapRange()
		// initialize default values
		var fg uint32 = 0
		var bg uint32 = 0
		var sp uint32 = 0
		reverse := false
		italic := false
		bold := false
		// strikethrough := false
		underline := false
		undercurl := false
		// blend := false
		// iterate over map and set attributes
		for mapIter.Next() {
			// TODO support strikethrough and blend
			key := mapIter.Key().String()
			switch key {
			case "foreground":
				fg = uint32(mapIter.Value().Elem().Convert(t).Uint())
				break
			case "background":
				bg = uint32(mapIter.Value().Elem().Convert(t).Uint())
				break
			case "special":
				sp = uint32(mapIter.Value().Elem().Convert(t).Uint())
				break
			// All boolean keys default to false,
			// and will only be sent when they are true.
			case "reverse":
				reverse = true
				break
			case "italic":
				italic = true
				break
			case "bold":
				bold = true
				break
			case "underline":
				underline = true
				break
			case "undercurl":
				undercurl = true
				break
			}
		}
		w.table.SetHlAttribute(int(id), fg, bg, sp, reverse, italic, bold, underline, undercurl)
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
			w.table.SetCell(row, &col_start, char, hl_id, repeat)
		}
	}
}

func grid_cursor_goto(w *Window, args []interface{}) {
	t := reflect.TypeOf(int(0))
	w.cursor.X = int(reflect.ValueOf(args).Index(0).Elem().Index(1).Elem().Convert(t).Int())
	w.cursor.Y = int(reflect.ValueOf(args).Index(0).Elem().Index(2).Elem().Convert(t).Int())
}

func grid_scroll(w *Window, args []interface{}) {
	t := reflect.TypeOf(int(0))
	top := reflect.ValueOf(args).Index(0).Elem().Index(1).Elem().Convert(t).Int()
	bot := reflect.ValueOf(args).Index(0).Elem().Index(2).Elem().Convert(t).Int()
	left := reflect.ValueOf(args).Index(0).Elem().Index(3).Elem().Convert(t).Int()
	right := reflect.ValueOf(args).Index(0).Elem().Index(4).Elem().Convert(t).Int()
	rows := reflect.ValueOf(args).Index(0).Elem().Index(5).Elem().Convert(t).Int()
	//cols := reflect.ValueOf(args).Index(0).Elem().Index(6).Elem().Convert(t).Int()
	w.table.Scroll(int(top), int(bot), int(rows), int(left), int(right))
}
