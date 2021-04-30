package main

import (
	"fmt"
	"time"

	rl "github.com/chunqian/go-raylib/raylib"
)

type Input struct {
	last_key                int32
	last_key_start_time     time.Time
	last_key_last_send_time time.Time

	ctrl_key  bool
	alt_key   bool
	shift_key bool

	// options
	hold_delay_begin        int
	hold_delay_between_keys int
}

var SpecialKeys = map[rl.KeyboardKey]string{
	rl.KEY_ESCAPE:    "ESC",
	rl.KEY_ENTER:     "CR",
	rl.KEY_KP_ENTER:  "CR",
	rl.KEY_SPACE:     "Space",
	rl.KEY_BACKSPACE: "BS",
	rl.KEY_UP:        "Up",
	rl.KEY_DOWN:      "Down",
	rl.KEY_RIGHT:     "Right",
	rl.KEY_LEFT:      "Left",
	rl.KEY_TAB:       "Tab",
	rl.KEY_INSERT:    "Insert",
	rl.KEY_DELETE:    "Del",
	rl.KEY_HOME:      "Home",
	rl.KEY_END:       "End",
	rl.KEY_PAGE_UP:   "PageUp",
	rl.KEY_PAGE_DOWN: "PageDown",
	rl.KEY_F1:        "F1",
	rl.KEY_F2:        "F2",
	rl.KEY_F3:        "F3",
	rl.KEY_F4:        "F4",
	rl.KEY_F5:        "F5",
	rl.KEY_F6:        "F6",
	rl.KEY_F7:        "F7",
	rl.KEY_F8:        "F8",
	rl.KEY_F9:        "F9",
	rl.KEY_F10:       "F10",
	rl.KEY_F11:       "F11",
	rl.KEY_F12:       "F12",
}

func (input *Input) HandleInputEvents(proc *NvimProcess) {
	keycode := input.get_keycode()
	if keycode != "" {
		// fmt.Println("Key:", keycode)
		term_code, err := proc.handle.ReplaceTermcodes(keycode, true, true, true)
		if err != nil {
			fmt.Println("ReplaceTermcodes:", err)
			return
		}
		err = proc.handle.FeedKeys(term_code, "m", false)
		if err != nil {
			fmt.Println("FeedKeys:", err)
		}
	}
}

func (input *Input) get_keycode() string {

	input.ctrl_key = rl.IsKeyDown(int32(rl.KEY_LEFT_CONTROL)) || rl.IsKeyDown(int32(rl.KEY_RIGHT_CONTROL))
	input.shift_key = rl.IsKeyDown(int32(rl.KEY_LEFT_SHIFT)) || rl.IsKeyDown(int32(rl.KEY_RIGHT_SHIFT))
	input.alt_key = rl.IsKeyDown(int32(rl.KEY_LEFT_ALT))

	keys := make([]string, 0)

	special_key := false
	multi_key := false

	if input.last_key != 0 {
		if rl.IsKeyDown(input.last_key) {
			// check key timing
			if int(time.Since(input.last_key_start_time).Milliseconds()) > input.hold_delay_begin &&
				int(time.Since(input.last_key_last_send_time).Milliseconds()) > input.hold_delay_between_keys {
				// check if this key is special
				if val, ok := SpecialKeys[rl.KeyboardKey(input.last_key)]; ok == true {
					keys = append(keys, val)
					special_key = true
				} else {
					input.last_key = 0
				}
				input.last_key_last_send_time = time.Now()
			}
		} else {
			input.last_key = 0
		}
	}

	for key := range SpecialKeys {
		if rl.IsKeyPressed(int32(key)) {
			if val, ok := SpecialKeys[rl.KeyboardKey(key)]; ok == true {
				keys = append(keys, val)
				special_key = true
				input.last_key = int32(key)
				input.last_key_start_time = time.Now()
				input.last_key_last_send_time = time.Now()
			}
		}
	}

	if (input.ctrl_key || input.alt_key) && !special_key {
		// NOTE: Raylib GetCharPressed doesn't return anything when alt or ctrl keys are holding.
		// Because of this, we will send keycode directly. This will broke other keyboard layouts,
		// but English layouts will send correctly. Only some specific keys will not work with modifier keys.
		for ckey := rl.GetKeyPressed(); ckey != 0; ckey = rl.GetKeyPressed() {
			keys = append(keys, string(ckey))
		}
	} else {
		for char := rl.GetCharPressed(); char != 0; char = rl.GetCharPressed() {
			// only send not evaluated keys
			if _, ok := SpecialKeys[rl.KeyboardKey(char)]; ok == false {
				keys = append(keys, string(char))
			}
		}
	}

	if len(keys) == 0 {
		// dont go further
		return ""
	}

	if input.shift_key && special_key {
		// Send shift if key is not character
		keys = append(keys, "S")
		// Shift is the first element now
		keys[0], keys[len(keys)-1] = keys[len(keys)-1], keys[0]
		multi_key = true
	}

	if input.alt_key {
		keys = append(keys, "A")
		keys[0], keys[len(keys)-1] = keys[len(keys)-1], keys[0]
		multi_key = true
	}

	if input.ctrl_key {
		keys = append(keys, "C")
		keys[0], keys[len(keys)-1] = keys[len(keys)-1], keys[0]
		multi_key = true
	}

	keycode := ""

	surround := multi_key || special_key

	if surround {
		keycode += "<"
	}

	for i, k := range keys {
		keycode += k
		if surround && i != len(keys)-1 {
			keycode += "-"
		}
	}

	if surround {
		keycode += ">"
	}

	return keycode
}
