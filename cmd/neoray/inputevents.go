package main

import (
	"strings"

	"github.com/go-gl/glfw/v3.3/glfw"
)

var (
	SpecialKeys = map[glfw.Key]string{
		glfw.KeyEscape:    "ESC",
		glfw.KeyEnter:     "CR",
		glfw.KeyKPEnter:   "CR",
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

	// Last mouse informations
	MousePos       IntVec2
	MouseButton    string
	MouseModifiers string
	MouseAction    glfw.Action
)

func InitializeInputEvents() {
	EditorSingleton.window.handle.SetCharModsCallback(CharEventHandler)
	EditorSingleton.window.handle.SetKeyCallback(KeyEventHandler)
	EditorSingleton.window.handle.SetMouseButtonCallback(ButtonEventHandler)
	EditorSingleton.window.handle.SetCursorPosCallback(MousePosEventHandler)
	EditorSingleton.window.handle.SetScrollCallback(ScrollEventHandler)
	EditorSingleton.window.handle.SetDropCallback(DropEventHandler)
}

func CharEventHandler(w *glfw.Window, char rune, mods glfw.ModifierKey) {
	var keycode string
	c := string(char)
	switch c {
	case " ":
		return
	case "<":
		keycode = "<LT>"
	default:
		keycode = c
	}
	EditorSingleton.nvim.SendKeyCode(keycode)
}

func KeyEventHandler(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	// TODO: We can simplify this
	if action == glfw.Press {
		if key == glfw.KeyLeftControl || key == glfw.KeyRightControl {
			if strings.Index(MouseModifiers, "C") == -1 {
				MouseModifiers += "C"
			}
		}
		if key == glfw.KeyLeftShift || key == glfw.KeyRightShift {
			if strings.Index(MouseModifiers, "S") == -1 {
				MouseModifiers += "S"
			}
		}
		if key == glfw.KeyLeftAlt || key == glfw.KeyRightAlt {
			if strings.Index(MouseModifiers, "A") == -1 {
				MouseModifiers += "A"
			}
		}
		if key == glfw.KeyLeftSuper || key == glfw.KeyRightSuper {
			if strings.Index(MouseModifiers, "D") == -1 {
				MouseModifiers += "D"
			}
		}
	} else if action == glfw.Release && MouseModifiers != "" {
		if key == glfw.KeyLeftControl || key == glfw.KeyRightControl {
			MouseModifiers = strings.Replace(MouseModifiers, "C", "", 1)
		}
		if key == glfw.KeyLeftShift || key == glfw.KeyRightShift {
			MouseModifiers = strings.Replace(MouseModifiers, "S", "", 1)
		}
		if key == glfw.KeyLeftAlt || key == glfw.KeyRightAlt {
			MouseModifiers = strings.Replace(MouseModifiers, "A", "", 1)
		}
		if key == glfw.KeyLeftSuper || key == glfw.KeyRightSuper {
			MouseModifiers = strings.Replace(MouseModifiers, "D", "", 1)
		}
	}

	if action != glfw.Release {
		ctrl := mods&glfw.ModControl != 0
		shift := mods&glfw.ModShift != 0
		alt := mods&glfw.ModAlt != 0
		super := mods&glfw.ModSuper != 0
		var keyname string
		name, ok := SpecialKeys[key]
		if ok {
			keyname = name
		} else {
			if shift || alt || super {
				return
			} else if ctrl {
				keyname = glfw.GetKeyName(key, scancode)
				if keyname == "" {
					return
				}
			}
		}
		keycode := "<"
		if ctrl {
			keycode += "C-"
		}
		if shift {
			keycode += "S-"
		}
		if alt {
			keycode += "A-"
		}
		if super {
			keycode += "D-"
		}
		keycode += keyname + ">"

		// Neoray keybindings are there.
		switch keycode {
		case "<F11>":
			EditorSingleton.window.ToggleFullscreen()
			return
		case "<ESC>":
			EditorSingleton.popupMenu.Hide()
			break
		default:
			break
		}

		EditorSingleton.nvim.SendKeyCode(keycode)
	}
}

func ButtonEventHandler(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	var buttonCode string
	switch button {
	case glfw.MouseButtonLeft:
		if action == glfw.Press {
			if EditorSingleton.popupMenu.MouseClick(false, MousePos) {
				return
			}
		}
		buttonCode = "left"
		break
	case glfw.MouseButtonRight:
		// We don't send right button to neovim.
		if action == glfw.Press {
			EditorSingleton.popupMenu.MouseClick(true, MousePos)
		}
		return
	case glfw.MouseButtonMiddle:
		buttonCode = "middle"
		break
	default:
		return
	}

	actionCode := "press"
	if action == glfw.Release {
		actionCode = "release"
	}

	row := MousePos.Y / EditorSingleton.cellHeight
	col := MousePos.X / EditorSingleton.cellWidth
	EditorSingleton.nvim.SendButton(buttonCode, actionCode, MouseModifiers, 0, row, col)

	MouseButton = buttonCode
	MouseAction = action
}

func MousePosEventHandler(w *glfw.Window, xpos, ypos float64) {
	MousePos.X = int(xpos)
	MousePos.Y = int(ypos)
	EditorSingleton.popupMenu.MouseMove(MousePos)
	// If mouse moving when holding left button, it's drag event
	if MouseAction == glfw.Press {
		row := MousePos.Y / EditorSingleton.cellHeight
		col := MousePos.X / EditorSingleton.cellWidth
		EditorSingleton.nvim.SendButton(MouseButton, "drag", MouseModifiers, 0, row, col)
	}
}

func ScrollEventHandler(w *glfw.Window, xpos, ypos float64) {
	action := "up"
	if ypos < 0 {
		action = "down"
	}
	row := MousePos.Y / EditorSingleton.cellHeight
	col := MousePos.X / EditorSingleton.cellWidth
	EditorSingleton.nvim.SendButton("wheel", action, MouseModifiers, 0, row, col)
}

func DropEventHandler(w *glfw.Window, names []string) {
	for _, name := range names {
		EditorSingleton.nvim.OpenFile(name)
	}
}
