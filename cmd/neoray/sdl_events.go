package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

var SpecialKeys = map[sdl.Keycode]string{
	sdl.K_ESCAPE:    "ESC",
	sdl.K_RETURN:    "CR",
	sdl.K_KP_ENTER:  "CR",
	sdl.K_SPACE:     "Space",
	sdl.K_BACKSPACE: "BS",
	sdl.K_UP:        "Up",
	sdl.K_DOWN:      "Down",
	sdl.K_RIGHT:     "Right",
	sdl.K_LEFT:      "Left",
	sdl.K_TAB:       "Tab",
	sdl.K_INSERT:    "Insert",
	sdl.K_DELETE:    "Del",
	sdl.K_HOME:      "Home",
	sdl.K_END:       "End",
	sdl.K_PAGEUP:    "PageUp",
	sdl.K_PAGEDOWN:  "PageDown",
	sdl.K_F1:        "F1",
	sdl.K_F2:        "F2",
	sdl.K_F3:        "F3",
	sdl.K_F4:        "F4",
	sdl.K_F5:        "F5",
	sdl.K_F6:        "F6",
	sdl.K_F7:        "F7",
	sdl.K_F8:        "F8",
	sdl.K_F9:        "F9",
	sdl.K_F10:       "F10",
	sdl.K_F11:       "F11",
	sdl.K_F12:       "F12",
}

var last_mouse_state uint8

func HandleSDLEvents(editor *Editor) {
	defer measure_execution_time("HandleSDLEvents")()
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			editor.quit_requested = true
			break
		case *sdl.DropEvent:
			break
		case *sdl.ClipboardEvent:
			break
		case *sdl.MouseButtonEvent, *sdl.MouseMotionEvent, *sdl.MouseWheelEvent:
			handle_mouse_event(&editor.nvim, event)
			break
		case *sdl.KeyboardEvent, *sdl.TextInputEvent:
			handle_input_event(&editor.nvim, event)
			break
		}
	}
}

func handle_mouse_event(nvim *NvimProcess, event sdl.Event) {
	var button string
	var action string
	var modifiers string
	var grid int
	var row int
	var col int
	switch t := event.(type) {
	case *sdl.MouseButtonEvent:
		switch t.Button {
		case sdl.BUTTON_LEFT:
			button = "left"
			break
		case sdl.BUTTON_RIGHT:
			button = "right"
			break
		case sdl.BUTTON_MIDDLE:
			button = "middle"
			break
		default:
			return
		}
		action = "press"
		if t.State == sdl.RELEASED {
			action = "release"
		}
		last_mouse_state = t.State
		row = int(t.Y) / GLOB_CellHeight
		col = int(t.X) / GLOB_CellWidth
		break
	case *sdl.MouseMotionEvent:
		if last_mouse_state == sdl.PRESSED {
			switch t.State {
			case sdl.BUTTON_LEFT:
				button = "left"
				break
			case sdl.BUTTON_RIGHT:
				button = "right"
				break
			case sdl.BUTTON_MIDDLE:
				button = "middle"
				break
			default:
				return
			}
			action = "drag"
			row = int(t.Y) / GLOB_CellHeight
			col = int(t.X) / GLOB_CellWidth
		}
		break
	case *sdl.MouseWheelEvent:
		button = "wheel"
		action = "up"
		if t.Y < 0 {
			action = "down"
		}
		row = int(t.Y) / GLOB_CellHeight
		col = int(t.X) / GLOB_CellWidth
		break
	}
	if button == "" || action == "" {
		return
	}
	nvim.SendButton(button, action, modifiers, grid, row, col)
}

func handle_input_event(nvim *NvimProcess, event sdl.Event) {
	var keys []string
	modifier_key := false
	special_key := false
	character_key := false

	// preprocess input keys and characters
	switch t := event.(type) {
	case *sdl.KeyboardEvent:
		if t.State == sdl.RELEASED {
			return
		}

		keys = make([]string, 0)
		get_char_with_modifier := false
		mod := t.Keysym.Mod
		if has_flag_u16(mod, sdl.KMOD_RALT) {
			// Dont do any combinations with right alt
		} else if has_flag_u16(mod, sdl.KMOD_CTRL) {
			keys = append(keys, "C")
			modifier_key = true
			get_char_with_modifier = true
		} else if has_flag_u16(mod, sdl.KMOD_LALT) {
			keys = append(keys, "A")
			modifier_key = true
			get_char_with_modifier = true
		} else if has_flag_u16(mod, sdl.KMOD_SHIFT) {
			keys = append(keys, "S")
			modifier_key = true
		} else if has_flag_u16(mod, sdl.KMOD_GUI) {
			keys = append(keys, "D")
			modifier_key = true
			get_char_with_modifier = true
		}

		if val, ok := SpecialKeys[t.Keysym.Sym]; ok == true {
			keys = append(keys, val)
			special_key = true
		} else if get_char_with_modifier && t.Keysym.Sym < 10000 {
			key := string(t.Keysym.Sym)
			keys = append(keys, key)
			character_key = true
		}
		break
	case *sdl.TextInputEvent:
		if !character_key { //&& t.GetText() != " " { // we are handling space in sdl.KeyboardEvent
			keys = append(keys, t.GetText())
			// log_debug_msg("Char:", t.GetText())
			character_key = true
		}
		break
	}

	// Prepare keycode to send neovim as neovim style
	keycode := ""
	if len(keys) == 0 || (modifier_key && !special_key && !character_key) {
		return
	} else if len(keys) == 1 && character_key {
		// This is special for neovim
		if keys[0] == "<" {
			keys[0] = "<LT>"
		}
		keycode = keys[0]
	} else {
		keycode += "<"
		for i, c := range keys {
			// This is special for neovim
			if c == "<" {
				c = "LT"
			}
			keycode += c
			if i != len(keys)-1 {
				keycode += "-"
			}
		}
		keycode += ">"
		// log_debug_msg("Key:", keycode)
	}

	// send keycode to neovim
	nvim.SendKeyCode(keycode)
}
