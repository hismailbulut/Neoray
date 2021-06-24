package neoray

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
	guifontwide   string
	linespace     int
	pumblend      int
	showtabline   int
	termguicolors bool
	// parsed options for forward usage
	guifontname string
	guifontsize float32
}

func (options *UIOptions) SetGuiFont(newGuiFont string) {
	// Load Font
	if newGuiFont != "" && newGuiFont != " " && newGuiFont != options.guifont {
		options.guifont = newGuiFont
		size := float32(DEFAULT_FONT_SIZE)
		// treat underlines like whitespaces
		newGuiFont = strings.ReplaceAll(newGuiFont, "_", " ")
		// parse font options
		fontOptions := strings.Split(newGuiFont, ":")
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
		if name == options.guifontname {
			// Names are same, just resize the font
			EditorSingleton.renderer.SetFontSize(size)
		} else {
			// Create and set renderers font.
			font, ok := CreateFont(name, size)
			if !ok {
				EditorSingleton.nvim.echoErr("Font %s not found!", name)
			} else {
				EditorSingleton.renderer.SetFont(font)
			}
		}
		options.guifontname = name
		options.guifontsize = size
	}
}