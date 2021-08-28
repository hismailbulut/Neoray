package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

const (
	ModControl BitMask = 1 << iota
	ModShift
	ModAlt
	ModSuper
	ModAltGr
)

var (
	SpecialKeys = map[glfw.Key]string{
		glfw.KeyEscape:    "ESC",
		glfw.KeyEnter:     "CR",
		glfw.KeyKPEnter:   "kEnter",
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

	SpecialChars = map[rune]string{
		'<':  "lt",
		'\\': "Bslash",
		'|':  "Bar",
		' ':  "Space",
	}

	SharedKeys = map[glfw.Key]struct {
		s string
		r rune
	}{
		glfw.KeyKPAdd:      {s: "kPlus", r: '+'},
		glfw.KeyKPSubtract: {s: "kMinus", r: '-'},
		glfw.KeyKPMultiply: {s: "kMultiply", r: '*'},
		glfw.KeyKPDivide:   {s: "kDivide", r: '/'},
		glfw.KeyKPDecimal:  {s: "kComma", r: ','},
		glfw.KeyKP0:        {s: "k0", r: '0'},
		glfw.KeyKP1:        {s: "k1", r: '1'},
		glfw.KeyKP2:        {s: "k2", r: '2'},
		glfw.KeyKP3:        {s: "k3", r: '3'},
		glfw.KeyKP4:        {s: "k4", r: '4'},
		glfw.KeyKP5:        {s: "k5", r: '5'},
		glfw.KeyKP6:        {s: "k6", r: '6'},
		glfw.KeyKP7:        {s: "k7", r: '7'},
		glfw.KeyKP8:        {s: "k8", r: '8'},
		glfw.KeyKP9:        {s: "k9", r: '9'},
	}

	// Global input variables
	lastMousePos    IntVec2
	lastMouseButton string
	lastMouseAction glfw.Action
	lastSharedKey   glfw.Key
	lastModifiers   BitMask
)

func initInputEvents() {
	// Initialize callbacks
	wh := singleton.window.handle
	wh.SetCharCallback(charCallback)
	wh.SetKeyCallback(keyCallback)
	wh.SetMouseButtonCallback(mouseButtonCallback)
	wh.SetCursorPosCallback(cursorPosCallback)
	wh.SetScrollCallback(scrollCallback)
	wh.SetDropCallback(dropCallback)
	logDebug("Input callbacks are initialized.")
}

func sendKeyInput(keycode string) {
	if !checkNeorayKeybindings(keycode) {
		singleton.nvim.input(keycode)
	}
}

func sendMouseInput(button, action string, mods BitMask, grid, row, column int) {
	// We need to create keycode from this parameters
	keycode := ""
	switch button {
	case "left":
		keycode = "Left"
	case "right":
		keycode = "Right"
	case "middle":
		keycode = "Middle"
	case "wheel":
		keycode = "ScrollWheel"
	default:
		panic("invalid mouse button")
	}
	switch action {
	case "press":
		keycode += "Mouse"
	case "drag":
		keycode += "Drag"
	case "release":
		keycode += "Release"
	case "up":
		assert(button == "wheel", "up mouse action without wheel")
		keycode += "Up"
	case "down":
		assert(button == "wheel", "down mouse action without wheel")
		keycode += "Down"
	default:
		panic("invalid mouse action")
	}
	keycode = "<" + modsStr(mods, true) + keycode + ">"
	logDebug("Mouse:", keycode)
	if !checkNeorayKeybindings(keycode) {
		singleton.nvim.inputMouse(button, action, modsStr(mods, false), grid, row, column)
	}
}

// Returns true if the key is emitted from neoray, and dont send it to neovim.
func checkNeorayKeybindings(keycode string) bool {
	// Handle neoray keybindings
	switch keycode {
	case singleton.options.keyIncreaseFontSize:
		singleton.renderer.increaseFontSize()
		return true
	case singleton.options.keyDecreaseFontSize:
		singleton.renderer.decreaseFontSize()
		return true
	case singleton.options.keyToggleFullscreen:
		singleton.window.toggleFullscreen()
		return true
	case "<ESC>":
		// Hide context menu if esc pressed.
		if singleton.options.contextMenuEnabled && !singleton.contextMenu.hidden {
			singleton.contextMenu.Hide()
			return true
		}
	}
	// Debugging only keybindings
	if isDebugBuild() {
		switch keycode {
		case "<C-F2>":
			panic("Control+F2 manual panic")
		case "<C-F3>":
			logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Control+F3 manual fatal")
		}
	}
	return false
}

// charCallback and keyCallback functions are separated because of the tests.

func charCallback(w *glfw.Window, char rune) {
	keycode := parseCharInput(char, lastModifiers)
	if keycode == "" {
		return
	}

	logDebug("Char:", keycode)
	sendKeyInput(keycode)

	if singleton.options.mouseHide {
		singleton.window.hideCursor()
	}
}

func parseCharInput(char rune, mods BitMask) string {
	shared, ok := SharedKeys[lastSharedKey]
	if ok && char == shared.r {
		lastSharedKey = glfw.KeyUnknown
		return ""
	}

	if mods.has(ModControl) || mods.has(ModAlt) {
		if !mods.has(ModAltGr) {
			return ""
		}
	}

	special, ok := SpecialChars[char]
	if ok {
		return "<" + modsStr(mods, true) + special + ">"
	} else {
		// Dont send S alone with char
		if mods.hasonly(ModShift) {
			mods.disable(ModShift)
		}
		if mods == 0 || mods.hasonly(ModAltGr) {
			return string(char)
		} else {
			return "<" + modsStr(mods, true) + string(char) + ">"
		}
	}
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// If this is a modifier key, we will store if it was pressed,
	// delete if it was released
	switch key {
	case glfw.KeyLeftAlt:
		lastModifiers.enableif(ModAlt, action != glfw.Release)
		return
	case glfw.KeyRightAlt:
		lastModifiers.enableif(ModAltGr, action != glfw.Release)
		return
	case glfw.KeyLeftControl, glfw.KeyRightControl:
		lastModifiers.enableif(ModControl, action != glfw.Release)
		return
	case glfw.KeyLeftShift, glfw.KeyRightShift:
		lastModifiers.enableif(ModShift, action != glfw.Release)
		return
	case glfw.KeyLeftSuper, glfw.KeyRightSuper:
		lastModifiers.enableif(ModSuper, action != glfw.Release)
		return
	}

	// Keys
	if action != glfw.Release {
		keycode := parseKeyInput(key, scancode, lastModifiers)
		if keycode == "" {
			return
		}

		logDebug("Key:", keycode)
		sendKeyInput(keycode)
	}
}

func parseKeyInput(key glfw.Key, scancode int, mods BitMask) string {
	if name, ok := SpecialKeys[key]; ok {
		// Send all combination with these keys because they dont produce a character.
		return "<" + modsStr(mods, true) + name + ">"
	} else if pair, ok := SharedKeys[key]; ok {
		// Shared keys are keypad keys and also all of them
		// are characters. They must be sent with their
		// special names for allowing more mappings. And
		// corresponding character mustn't be sent.
		lastSharedKey = key
		return "<" + modsStr(mods, true) + pair.s + ">"
	} else if mods != 0 && !mods.has(ModAltGr) && !mods.hasonly(ModShift) {
		// Only send if there is modifiers
		// Dont send with altgr
		// Dont send shift alone

		// Do not send if key is unknown and scancode is 0 because glfw panics.
		if key == glfw.KeyUnknown && scancode == 0 {
			return ""
		}

		// GetKeyName function returns the localized character
		// of the key if key representable by char. Ctrl with alt
		// means AltGr and it is used for alternative characters.
		// And shift is also changes almost every key.
		keyname := glfw.GetKeyName(key, scancode)
		if keyname != "" {
			return "<" + modsStr(mods, true) + keyname + ">"
		}
	}
	return ""
}

func mouseButtonCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if singleton.options.mouseHide {
		singleton.window.showCursor()
	}

	var buttonCode string

	switch button {
	case glfw.MouseButtonLeft:
		if action == glfw.Press && singleton.options.contextMenuEnabled {
			if singleton.contextMenu.mouseClick(false, lastMousePos) {
				return
			}
		}
		buttonCode = "left"
		break
	case glfw.MouseButtonRight:
		if action == glfw.Press && singleton.options.contextMenuEnabled {
			// We don't send right button to neovim if popup menu enabled.
			singleton.contextMenu.mouseClick(true, lastMousePos)
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
			singleton.debugPrintCell(lastMousePos)
		}
		return
	}

	actionCode := "press"
	if action == glfw.Release {
		actionCode = "release"
	}

	grid, row, col := singleton.gridManager.getCellAt(lastMousePos)
	sendMouseInput(buttonCode, actionCode, lastModifiers, grid, row, col)

	lastMouseButton = buttonCode
	lastMouseAction = action
}

func cursorPosCallback(w *glfw.Window, xpos, ypos float64) {
	if singleton.options.mouseHide {
		singleton.window.showCursor()
	}

	lastMousePos.X = int(xpos)
	lastMousePos.Y = int(ypos)

	if singleton.options.contextMenuEnabled {
		singleton.contextMenu.mouseMove(lastMousePos)
	}

	// If mouse moving when holding left button, it's drag event
	if lastMouseAction == glfw.Press {
		grid, row, col := singleton.gridManager.getCellAt(lastMousePos)
		sendMouseInput(lastMouseButton, "drag", lastModifiers, grid, row, col)
	}
}

func scrollCallback(w *glfw.Window, xpos, ypos float64) {
	if singleton.options.mouseHide {
		singleton.window.showCursor()
	}

	action := "up"
	if ypos < 0 {
		action = "down"
	}

	grid, row, col := singleton.gridManager.getCellAt(lastMousePos)
	sendMouseInput("wheel", action, lastModifiers, grid, row, col)
}

func dropCallback(w *glfw.Window, names []string) {
	for _, name := range names {
		singleton.nvim.openFile(name)
	}
}

func modsStr(mods BitMask, separated bool) string {
	str := ""
	if mods.has(ModAlt) {
		str += "M"
		if separated {
			str += "-"
		}
	}
	if mods.has(ModControl) {
		str += "C"
		if separated {
			str += "-"
		}
	}
	if mods.has(ModShift) {
		str += "S"
		if separated {
			str += "-"
		}
	}
	if mods.has(ModSuper) {
		str += "D"
		if separated {
			str += "-"
		}
	}
	return str
}
