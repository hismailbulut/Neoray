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

func HandleSDLEvents(editor *Editor) {
	defer measure_execution_time("HandleSDLEvents")()
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch event.(type) {
		case *sdl.QuitEvent:
			editor.quit_requested = true
		case *sdl.DropEvent:
		case *sdl.ClipboardEvent:
		case *sdl.KeyboardEvent, *sdl.TextInputEvent:
			handle_input_event(&editor.nvim, event)
		}
	}
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

	case *sdl.TextInputEvent:
		if !character_key && t.GetText() != " " { // we are handling space in sdl.KeyboardEvent
			keys = append(keys, t.GetText())
			// log_debug_msg("Char:", t.GetText())
			character_key = true
		}
	}

	// Prepare keycode to send neovim as neovim style
	keycode := ""
	if len(keys) == 0 || (modifier_key && !special_key && !character_key) {
		return
	} else if len(keys) == 1 && character_key {
		keycode = keys[0]
	} else {
		keycode += "<"
		for i, c := range keys {
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
