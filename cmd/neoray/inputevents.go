package main

import (
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type StringPair struct {
	key string
	val string
}

var (
	SpecialKeys = map[glfw.Key]string{
		glfw.KeyEscape:    "ESC",
		glfw.KeyEnter:     "CR",
		glfw.KeyKPEnter:   "kEnter",
		glfw.KeySpace:     "Space",
		glfw.KeyBackspace: "BS",
		glfw.KeyUp:        "Up",
		glfw.KeyDown:      "Down",
		glfw.KeyRight:     "Right",
		glfw.KeyLeft:      "Left",
		glfw.KeyTab:       "Tab",
		glfw.KeyInsert:    "Insert",
		glfw.KeyDelete:    "Del",
		glfw.KeyHome:      "Home",
		glfw.KeyEnd:       "End",
		glfw.KeyPageUp:    "PageUp",
		glfw.KeyPageDown:  "PageDown",
		glfw.KeyF1:        "F1",
		glfw.KeyF2:        "F2",
		glfw.KeyF3:        "F3",
		glfw.KeyF4:        "F4",
		glfw.KeyF5:        "F5",
		glfw.KeyF6:        "F6",
		glfw.KeyF7:        "F7",
		glfw.KeyF8:        "F8",
		glfw.KeyF9:        "F9",
		glfw.KeyF10:       "F10",
		glfw.KeyF11:       "F11",
		glfw.KeyF12:       "F12",
	}

	SpecialChars = map[string]string{
		"<":  "lt",
		"\\": "Bslash",
		"|":  "Bar",
		// These needs to be checked for surrounding with <>
		">": ">",
		"{": "{",
		"[": "[",
		"]": "]",
		"}": "}",
	}

	// Don't add right alt.
	ModifierKeys = map[glfw.Key]string{
		glfw.KeyLeftAlt:      "M",
		glfw.KeyLeftControl:  "C",
		glfw.KeyRightControl: "C",
		glfw.KeyLeftShift:    "S",
		glfw.KeyRightShift:   "S",
		glfw.KeyLeftSuper:    "D",
		glfw.KeyRightSuper:   "D",
	}

	SharedKeys = map[glfw.Key]StringPair{
		glfw.KeyKPAdd:      {key: "kPlus", val: "+"},
		glfw.KeyKPSubtract: {key: "kMinus", val: "-"},
		glfw.KeyKPMultiply: {key: "kMultiply", val: "*"},
		glfw.KeyKPDivide:   {key: "kDivide", val: "/"},
		glfw.KeyKPDecimal:  {key: "kComma", val: ","},
		glfw.KeyKP0:        {key: "k0", val: "0"},
		glfw.KeyKP1:        {key: "k1", val: "1"},
		glfw.KeyKP2:        {key: "k2", val: "2"},
		glfw.KeyKP3:        {key: "k3", val: "3"},
		glfw.KeyKP4:        {key: "k4", val: "4"},
		glfw.KeyKP5:        {key: "k5", val: "5"},
		glfw.KeyKP6:        {key: "k6", val: "6"},
		glfw.KeyKP7:        {key: "k7", val: "7"},
		glfw.KeyKP8:        {key: "k8", val: "8"},
		glfw.KeyKP9:        {key: "k9", val: "9"},
	}

	// Global input informations
	lastMousePos     IntVec2
	lastMouseButton  string
	lastMouseAction  glfw.Action
	lastSharedKey    glfw.Key
	currentModifiers string
)

func InitializeInputEvents() {
	// Initialize callbacks
	wh := EditorSingleton.window.handle
	wh.SetCharCallback(CharEventHandler)
	wh.SetKeyCallback(KeyEventHandler)
	wh.SetMouseButtonCallback(ButtonEventHandler)
	wh.SetCursorPosCallback(MousePosEventHandler)
	wh.SetScrollCallback(ScrollEventHandler)
	wh.SetDropCallback(DropEventHandler)
}

func CharEventHandler(w *glfw.Window, char rune) {
	var keycode string
	c := string(char)

	if c == " " {
		return
	}

	pair, ok := SharedKeys[lastSharedKey]
	if ok {
		if c == pair.val {
			lastSharedKey = glfw.KeyUnknown
			return
		}
	}

	sp, ok := SpecialChars[c]
	if ok {
		// Send modifiers with special characters. Like C-\
		if strings.Contains(currentModifiers, "M") {
			keycode += "M-"
		}
		if strings.Contains(currentModifiers, "C") {
			keycode += "C-"
		}
		if strings.Contains(currentModifiers, "D") {
			keycode += "D-"
		}
		if sp != c || len(keycode) > 0 {
			keycode = "<" + keycode + sp + ">"
		} else {
			keycode = sp
		}
	} else {
		keycode = c
	}

	EditorSingleton.nvim.input(keycode)

	if EditorSingleton.options.mouseHide {
		EditorSingleton.window.hideCursor()
	}
}

func KeyEventHandler(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// If this is a modifier key, we will store if it was pressed,
	// delete if it was released
	code, ok := ModifierKeys[key]
	if ok {
		if action == glfw.Press {
			if !strings.Contains(currentModifiers, code) {
				currentModifiers += code
			}
		} else if action == glfw.Release {
			currentModifiers = strings.Replace(currentModifiers, code, "", 1)
		}
		return
	}

	// Keys
	if action != glfw.Release {
		alt := mods&glfw.ModAlt != 0
		ctrl := mods&glfw.ModControl != 0
		shift := mods&glfw.ModShift != 0
		super := mods&glfw.ModSuper != 0

		var keyname string
		if name, ok := SpecialKeys[key]; ok {
			keyname = name
		} else if pair, ok := SharedKeys[key]; ok {
			// Maybe a shared key?
			// Shared keys are keypad keys and most of them
			// are characters. They must be sent with their
			// special names for allowing more mappings. And
			// corresponding character mustn't be sent.
			keyname = pair.key
			lastSharedKey = key
		} else if (ctrl || shift || alt || super) && !(ctrl && (alt || shift)) && !(shift && !(alt || super)) {
			// Only send if there is modifiers
			// Dont send ctrl with alt or shift
			// Dont send shift alone
			// These are the possible modifiers for characters:
			// M, D, C, M-D, C-D, M-S, S-D, M-S-D

			// GetKeyName function returns the localized character
			// of the key if key representable by char. Ctrl with alt
			// means AltGr and it is used for alternative characters.
			// And shift is also changes almost every key.
			keyname = glfw.GetKeyName(key, scancode)
			if keyname == "" {
				return
			}
		} else {
			return
		}

		keycode := "<"
		if alt {
			keycode += "M-"
		}
		if ctrl {
			keycode += "C-"
		}
		if shift {
			keycode += "S-"
		}
		if super {
			keycode += "D-"
		}
		keycode += keyname + ">"

		// Handle neoray keybindings
		switch keycode {
		case EditorSingleton.options.keyIncreaseFontSize:
			EditorSingleton.renderer.increaseFontSize()
			return
		case EditorSingleton.options.keyDecreaseFontSize:
			EditorSingleton.renderer.decreaseFontSize()
			return
		case EditorSingleton.options.keyToggleFullscreen:
			EditorSingleton.window.toggleFullscreen()
			return
		case "<ESC>":
			if EditorSingleton.options.popupMenuEnabled && !EditorSingleton.popupMenu.hidden {
				EditorSingleton.popupMenu.Hide()
				return
			}
			break
		}

		EditorSingleton.nvim.input(keycode)
	}
}

func ButtonEventHandler(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if EditorSingleton.options.mouseHide {
		EditorSingleton.window.showCursor()
	}
	var buttonCode string
	switch button {
	case glfw.MouseButtonLeft:
		if action == glfw.Press && EditorSingleton.options.popupMenuEnabled {
			if EditorSingleton.popupMenu.mouseClick(false, lastMousePos) {
				return
			}
		}
		buttonCode = "left"
		break
	case glfw.MouseButtonRight:
		if action == glfw.Press && EditorSingleton.options.popupMenuEnabled {
			// We don't send right button to neovim if popup menu enabled.
			EditorSingleton.popupMenu.mouseClick(true, lastMousePos)
			return
		}
		buttonCode = "right"
		break
	case glfw.MouseButtonMiddle:
		buttonCode = "middle"
		break
	default:
		// Other mouse buttons will print the cell info under the cursor in debug build.
		if isDebugBuild() && action == glfw.Release {
			row := lastMousePos.Y / EditorSingleton.cellHeight
			col := lastMousePos.X / EditorSingleton.cellWidth
			EditorSingleton.debugEvalCell(row, col)
		}
		return
	}

	actionCode := "press"
	if action == glfw.Release {
		actionCode = "release"
	}

	row := lastMousePos.Y / EditorSingleton.cellHeight
	col := lastMousePos.X / EditorSingleton.cellWidth
	EditorSingleton.nvim.inputMouse(buttonCode, actionCode, currentModifiers, 0, row, col)

	lastMouseButton = buttonCode
	lastMouseAction = action
}

func MousePosEventHandler(w *glfw.Window, xpos, ypos float64) {
	if EditorSingleton.options.mouseHide {
		EditorSingleton.window.showCursor()
	}
	lastMousePos.X = int(xpos)
	lastMousePos.Y = int(ypos)
	if EditorSingleton.options.popupMenuEnabled {
		EditorSingleton.popupMenu.mouseMove(lastMousePos)
	}
	// If mouse moving when holding left button, it's drag event
	if lastMouseAction == glfw.Press {
		row := lastMousePos.Y / EditorSingleton.cellHeight
		col := lastMousePos.X / EditorSingleton.cellWidth
		EditorSingleton.nvim.inputMouse(lastMouseButton, "drag", currentModifiers, 0, row, col)
	}
}

func ScrollEventHandler(w *glfw.Window, xpos, ypos float64) {
	if EditorSingleton.options.mouseHide {
		EditorSingleton.window.showCursor()
	}
	action := "up"
	if ypos < 0 {
		action = "down"
	}
	row := lastMousePos.Y / EditorSingleton.cellHeight
	col := lastMousePos.X / EditorSingleton.cellWidth
	EditorSingleton.nvim.inputMouse("wheel", action, currentModifiers, 0, row, col)
}

func DropEventHandler(w *glfw.Window, names []string) {
	for _, name := range names {
		EditorSingleton.nvim.openFile(name)
	}
}
