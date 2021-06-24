package neoray

import (
	"reflect"
)

func HandleNvimRedrawEvents() {
	defer measure_execution_time("HandleNvimRedrawEvents")()

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
				EditorSingleton.window.SetTitle(title)
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
				name := reflect.ValueOf(update[1]).Index(0).Elem().String()
				EditorSingleton.mode.current_mode_name = name
				id := reflect.ValueOf(update[1]).Index(1).Elem().Convert(reflect.TypeOf(int(0))).Int()
				EditorSingleton.mode.current_mode = int(id)
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
				break
			case "visual_bell":
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
				EditorSingleton.grid.ClearCells()
				break
			case "grid_destroy":
				break
			case "grid_cursor_goto":
				grid_cursor_goto(update[1:])
				break
			case "grid_scroll":
				grid_scroll(update[1:])
				break
			}
		}
	}
	// clear update stack
	EditorSingleton.nvim.update_stack = nil
}

func option_set(args []interface{}) {
	t := reflect.TypeOf(int(0))
	options := &EditorSingleton.options
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
			options.SetGuiFont(valr.String())
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

func mode_info_set(args []interface{}) {
	r := reflect.ValueOf(args).Index(0).Elem()
	EditorSingleton.mode.cursor_style_enabled = r.Index(0).Elem().Bool()
	t := reflect.TypeOf(int(0))
	EditorSingleton.mode.Clear()
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
		EditorSingleton.mode.Add(info)
	}
	EditorSingleton.cursor.needsDraw = true
}

func grid_resize(args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(int(0))
	rows := int(r.Index(2).Elem().Convert(t).Int())
	cols := int(r.Index(1).Elem().Convert(t).Int())

	EditorSingleton.rowCount = rows
	EditorSingleton.columnCount = cols
	EditorSingleton.cellCount = rows * cols

	EditorSingleton.grid.Resize(rows, cols)

	log_debug("Grid resized:", rows, cols)
	EditorSingleton.renderer.resize(rows, cols)

	EditorSingleton.waitingResize = false
}

func default_colors_set(args []interface{}) {
	r := reflect.ValueOf(args[0])
	t := reflect.TypeOf(uint(0))
	fg := r.Index(0).Elem().Convert(t).Uint()
	bg := r.Index(1).Elem().Convert(t).Uint()
	sp := r.Index(2).Elem().Convert(t).Uint()
	EditorSingleton.grid.default_fg = unpackColor(uint32(fg))
	EditorSingleton.grid.default_bg = unpackColor(uint32(bg))
	EditorSingleton.grid.default_sp = unpackColor(uint32(sp))
}

func hl_attr_define(args []interface{}) {
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
		hl_attr := HighlightAttribute{}
		// iterate over map and set attributes
		for mapIter.Next() {
			switch mapIter.Key().String() {
			case "foreground":
				fg := uint32(mapIter.Value().Elem().Convert(t).Uint())
				hl_attr.foreground = unpackColor(fg)
				break
			case "background":
				bg := uint32(mapIter.Value().Elem().Convert(t).Uint())
				hl_attr.background = unpackColor(bg)
				break
			case "special":
				sp := uint32(mapIter.Value().Elem().Convert(t).Uint())
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
				hl_attr.blend = int(mapIter.Value().Elem().Convert(t).Uint())
				break
			}
		}
		EditorSingleton.grid.attributes[id] = hl_attr
		EditorSingleton.grid.MakeAllCellsChanged()
	}
}

func grid_line(args []interface{}) {
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
			char := reflect.ValueOf(cell_slice).Index(0).Elem().String()
			repeat := 0
			if len(cell_slice) >= 2 {
				hl_id = int(reflect.ValueOf(cell_slice).Index(1).Elem().Convert(t).Int())
			}
			if len(cell_slice) == 3 {
				repeat = int(reflect.ValueOf(cell_slice).Index(2).Elem().Convert(t).Int())
			}
			EditorSingleton.grid.SetCell(row, &col_start, char, hl_id, repeat)
		}
	}
}

func grid_cursor_goto(args []interface{}) {
	t := reflect.TypeOf(int(0))
	r := reflect.ValueOf(args).Index(0).Elem()
	X := int(r.Index(1).Elem().Convert(t).Int())
	Y := int(r.Index(2).Elem().Convert(t).Int())
	EditorSingleton.cursor.SetPosition(X, Y, false)
}

func grid_scroll(args []interface{}) {
	t := reflect.TypeOf(int(0))
	r := reflect.ValueOf(args).Index(0).Elem()
	top := r.Index(1).Elem().Convert(t).Int()
	bot := r.Index(2).Elem().Convert(t).Int()
	left := r.Index(3).Elem().Convert(t).Int()
	right := r.Index(4).Elem().Convert(t).Int()
	rows := r.Index(5).Elem().Convert(t).Int()
	//cols := r.Index(6).Elem().Convert(t).Int()
	EditorSingleton.grid.Scroll(int(top), int(bot), int(rows), int(left), int(right))
}
