package main

import (
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/hismailbulut/neoray/src/common"
	"github.com/hismailbulut/neoray/src/logger"
)

const (
	ModControl common.BitMask = 1 << iota
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
	}

	SharedKeys = map[glfw.Key]struct {
		s string
		r rune
	}{
		glfw.KeySpace:      {s: "Space", r: ' '},
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
	inputCache struct {
		lastMousePos    common.Vector2[int]
		lastDragPos     common.Vector2[int]
		lastDragGrid    int
		lastMouseButton string
		lastMouseAction glfw.Action
		lastSharedKey   glfw.Key
		lastModifiers   common.BitMask
	}
)

func sendKeyInput(keycode string) {
	if !checkNeorayKeybindings(keycode) {
		Editor.nvim.input(keycode)
	}
}

func sendMouseInput(button, action string, mods common.BitMask, grid, row, column int) {
	// We need to create keycode from this parameters for
	// checking the mouse keybindings
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
		keycode += "Up"
	case "down":
		keycode += "Down"
	default:
		panic("invalid mouse action")
	}
	keycode = "<" + modsStr(mods) + keycode + ">"
	if !checkNeorayKeybindings(keycode) {
		Editor.nvim.inputMouse(button, action, modsStr(mods), grid, row, column)
	}
}

// Returns true if the key is emitted from neoray, and dont send it to neovim.
func checkNeorayKeybindings(keycode string) bool {
	// Handle neoray keybindings
	switch keycode {
	case Editor.options.keyIncreaseFontSize:
		Editor.gridManager.AddGridFontSize(1, 0.5)
		Editor.contextMenu.AddFontSize(0.5)
		return true
	case Editor.options.keyDecreaseFontSize:
		Editor.gridManager.AddGridFontSize(1, -0.5)
		Editor.contextMenu.AddFontSize(-0.5)
		return true
	case Editor.options.keyToggleFullscreen:
		Editor.window.ToggleFullscreen()
		return true
	case "<ESC>":
		// Hide context menu if esc pressed.
		if Editor.options.contextMenuEnabled && !Editor.contextMenu.hidden {
			Editor.contextMenu.Hide()
			return true
		}
	}
	// Debugging only keybindings
	if BUILD_TYPE == logger.Debug {
		switch keycode {
		case "<C-F2>":
			panic("Control+F2 manual panic")
		case "<C-F3>":
			logger.Log(logger.FATAL, "Control+F3 manual fatal")
		case "<C-F4>":
			toggle_cpu_profile()
			return true
		case "<C-F5>":
			dump_heap_profile()
			return true
		case "<MiddleMouse>":
			Editor.gridManager.printCellInfoAt(inputCache.lastMousePos)
			return true
		}
	}
	return false
}

func CharInputHandler(char rune) {
	keycode := parseCharInput(char, inputCache.lastModifiers)
	if keycode != "" {
		sendKeyInput(keycode)
		// Hide mouse if mousehide option set
		if Editor.uiOptions.mousehide {
			Editor.window.HideMouseCursor()
		}
	}
}

func parseCharInput(char rune, mods common.BitMask) string {
	shared, ok := SharedKeys[inputCache.lastSharedKey]
	if ok && char == shared.r {
		inputCache.lastSharedKey = glfw.KeyUnknown
		return ""
	}

	if mods.Has(ModControl) || mods.Has(ModAlt) {
		if !mods.Has(ModAltGr) {
			return ""
		}
	}

	// Dont send S alone with any char
	if mods.HasOnly(ModShift) {
		mods.Disable(ModShift)
	}

	special, ok := SpecialChars[char]
	if ok {
		return "<" + modsStr(mods) + special + ">"
	} else {
		if mods == 0 || mods.HasOnly(ModAltGr) {
			return string(char)
		} else {
			return "<" + modsStr(mods) + string(char) + ">"
		}
	}
}

func KeyInputHandler(key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {

	// Toggle modifiers
	switch key {
	case glfw.KeyLeftAlt:
		inputCache.lastModifiers.EnableIf(ModAlt, action != glfw.Release)
		return
	case glfw.KeyRightAlt:
		inputCache.lastModifiers.EnableIf(ModAltGr, action != glfw.Release)
		return
	case glfw.KeyLeftControl, glfw.KeyRightControl:
		inputCache.lastModifiers.EnableIf(ModControl, action != glfw.Release)
		return
	}

	// NOTE: For shift and super we dont need to look exact keypress, but for ctrl and alt we need to check Altgr
	// PROBLEM
	// 	Mods always contains one modifier, but there may be more than one for one modifier
	// 	Eg: Altgr generates Ctrl + Alt and user holding Ctrl, there must be two ctrl's, but it's not possible.
	// 	HACK: use reported system modifiers when altgr is not pressed, but we are checking exact keypress for altgr
	// 	and this also can be a problem.
	// 	Altgr is always a problem, why it's not a different mod?

	inputCache.lastModifiers.EnableIf(ModShift, action != glfw.Release && mods&glfw.ModShift != 0)
	inputCache.lastModifiers.EnableIf(ModSuper, action != glfw.Release && mods&glfw.ModSuper != 0)

	// Check is the modifiers are correct
	if (inputCache.lastModifiers.Has(ModAlt) != (mods&glfw.ModAlt != 0)) || (inputCache.lastModifiers.Has(ModControl) != (mods&glfw.ModControl != 0)) {
		// Use mods when altgr is disabled
		if !inputCache.lastModifiers.Has(ModAltGr) {
			inputCache.lastModifiers.EnableIf(ModAlt, action != glfw.Release && mods&glfw.ModAlt != 0)
			inputCache.lastModifiers.EnableIf(ModControl, action != glfw.Release && mods&glfw.ModControl != 0)
		}
	}

	// Keys
	if action != glfw.Release {
		keycode := parseKeyInput(key, scancode, inputCache.lastModifiers)
		if keycode != "" {
			sendKeyInput(keycode)
		}
	}
}

func parseKeyInput(key glfw.Key, scancode int, mods common.BitMask) string {
	if name, ok := SpecialKeys[key]; ok {
		// Send all combination with these keys because they dont produce a character.
		// We need to also enable altgr key, which means if altgr is pressed then we act like Ctrl+Alt pressed
		if mods.Has(ModAltGr) {
			mods.Enable(ModControl | ModAlt)
		}
		return "<" + modsStr(mods) + name + ">"
	} else if pair, ok := SharedKeys[key]; ok {
		// Shared keys are keypad keys and also all of them
		// are characters. They must be sent with their
		// special names for allowing more mappings. And
		// corresponding character mustn't be sent.
		inputCache.lastSharedKey = key
		// Do same thing above
		if mods.Has(ModAltGr) {
			mods.Enable(ModControl | ModAlt)
		}
		return "<" + modsStr(mods) + pair.s + ">"
	} else if mods != 0 && !mods.Has(ModAltGr) && !mods.HasOnly(ModShift) {
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
			return "<" + modsStr(mods) + keyname + ">"
		}
	}
	return ""
}

func MouseInputHandler(button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	// Show mouse when mouse button pressed
	if Editor.uiOptions.mousehide {
		Editor.window.ShowMouseCursor()
	}

	var buttonCode string
	switch button {
	case glfw.MouseButtonLeft:
		if action == glfw.Press && Editor.options.contextMenuEnabled {
			if Editor.contextMenu.MouseClick(false, inputCache.lastMousePos) {
				// Mouse clicked to context menu, dont send to neovim.
				// TODO: We also need to dont send release action to neovim.
				return
			}
		}
		buttonCode = "left"
		break
	case glfw.MouseButtonRight:
		// We don't send right button to neovim if popup menu enabled.
		if Editor.options.contextMenuEnabled {
			if action == glfw.Press {
				Editor.contextMenu.MouseClick(true, inputCache.lastMousePos)
			}
			return
		}
		buttonCode = "right"
		break
	case glfw.MouseButtonMiddle:
		buttonCode = "middle"
		break
	default:
		// Other mouse buttons will print the cell info under the cursor in debug build.
		// if isDebugBuild() && action == glfw.Release {
		//     Editor.debugPrintCell(inputCache.lastMousePos)
		// }
		return
	}

	actionCode := "press"
	if action == glfw.Release {
		actionCode = "release"
	}

	grid, row, col := Editor.gridManager.CellAt(inputCache.lastMousePos)
	sendMouseInput(buttonCode, actionCode, inputCache.lastModifiers, grid, row, col)

	inputCache.lastMouseButton = buttonCode
	inputCache.lastMouseAction = action
}

func MouseMoveHandler(xpos, ypos float64) {
	// Show mouse when mouse moved
	if Editor.uiOptions.mousehide {
		Editor.window.ShowMouseCursor()
	}

	inputCache.lastMousePos.X = int(xpos)
	inputCache.lastMousePos.Y = int(ypos)

	if Editor.options.contextMenuEnabled {
		Editor.contextMenu.MouseMove(inputCache.lastMousePos)
	}

	// If mouse moving when holding button, it's a drag event
	if inputCache.lastMouseAction == glfw.Press {
		grid, row, col := Editor.gridManager.CellAt(inputCache.lastMousePos)
		// NOTE: Drag event as some multigrid issues
		// Sending drag event on same row and column causes whole word is selected
		if grid != inputCache.lastDragGrid || row != inputCache.lastDragPos.X || col != inputCache.lastDragPos.Y {
			sendMouseInput(inputCache.lastMouseButton, "drag", inputCache.lastModifiers, grid, row, col)
			inputCache.lastDragGrid = grid
			inputCache.lastDragPos.X = row
			inputCache.lastDragPos.Y = col
		}
	}
}

func ScrollHandler(xoff, yoff float64) {
	if Editor.uiOptions.mousehide {
		Editor.window.ShowMouseCursor()
	}

	action := "up"
	if yoff < 0 {
		action = "down"
	}

	grid, row, col := Editor.gridManager.CellAt(inputCache.lastMousePos)
	sendMouseInput("wheel", action, inputCache.lastModifiers, grid, row, col)
}

func DropHandler(names []string) {
	for _, name := range names {
		Editor.nvim.openFile(name)
	}
}

func modsStr(mods common.BitMask) string {
	str := ""
	if mods.Has(ModAlt) {
		str += "M-"
	}
	if mods.Has(ModControl) {
		str += "C-"
	}
	if mods.Has(ModShift) {
		str += "S-"
	}
	if mods.Has(ModSuper) {
		str += "D-"
	}
	return str
}
