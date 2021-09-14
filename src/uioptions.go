package main

import (
	"strconv"
	"strings"
)

type UIOptions struct {
	// neovim options
	arabicshape   bool
	ambiwidth     string
	emoji         bool
	guifont       string
	guifontset    string
	guifontwide   string // TODO
	linespace     int    // TODO
	pumblend      int    // TODO
	showtabline   int
	termguicolors bool
	// will be implemented soon, currently always true
	mousehide bool
	// parsed options for forward usage
	parsed struct {
		guifontname string
		guifontsize float32
	}
}

func CreateUIOptions() UIOptions {
	return UIOptions{
		mousehide: true,
	}
}

func (options *UIOptions) setGuiFont(guifont string) {
	// Load Font
	if guifont != options.guifont {
		options.guifont = guifont
		var size float32 = DEFAULT_FONT_SIZE
		// treat underlines like whitespaces
		guifont = strings.ReplaceAll(guifont, "_", " ")
		// parse font options
		fontOptions := strings.Split(guifont, ":")
		name := fontOptions[0]
		for _, opt := range fontOptions[1:] {
			if len(opt) > 1 && opt[0] == 'h' {
				// Font size
				tsize, err := strconv.ParseFloat(opt[1:], 32)
				if err == nil {
					size = float32(tsize)
				}
			}
		}
		if name == "" {
			// Disable user font.
			singleton.renderer.disableUserFont()
			singleton.renderer.setFontSize(size)
		} else if name == options.parsed.guifontname {
			// Names are same, just resize the font
			singleton.renderer.setFontSize(size)
		} else {
			// Create and set renderers font.
			font, ok := CreateFont(name, size)
			if ok {
				singleton.renderer.setFont(font)
			} else {
				logMessage(LEVEL_ERROR, TYPE_NEORAY, "Font", name, "not found!")
				singleton.nvim.echoErr("Font %s not found!", name)
			}
		}
		options.parsed.guifontname = name
		options.parsed.guifontsize = size
	}
}
