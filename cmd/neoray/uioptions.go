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
	guifontwide   string
	linespace     int
	pumblend      int
	showtabline   int
	termguicolors bool
	// parsed options for forward usage
	guifontname string
}

func (options *UIOptions) SetGuiFont(newGuiFont string) {
	// Load Font
	if newGuiFont != "" && newGuiFont != " " && newGuiFont != options.guifont {
		options.guifont = newGuiFont
		size := DEFAULT_FONT_SIZE
		// treat underlines like whitespaces
		newGuiFont = strings.ReplaceAll(newGuiFont, "_", " ")
		// parse font options
		fontOptions := strings.Split(newGuiFont, ":")
		name := fontOptions[0]
		for _, opt := range fontOptions[1:] {
			if opt[0] == 'h' {
				tsize, err := strconv.Atoi(opt[1:])
				if err == nil {
					size = tsize
				}
			}
		}
		if name == systemFontDefault {
			// If this is the system default font, we already have loaded
			EditorSingleton.renderer.DisableUserFont()
			EditorSingleton.renderer.SetFontSize(size)
		} else if name == options.guifontname {
			// Names are same, just resize the font
			EditorSingleton.renderer.SetFontSize(size)
		} else {
			// Create and set renderers font.
			font, ok := CreateFont(name, size)
			if !ok {
				return
			}
			EditorSingleton.renderer.SetFont(font)
		}
		options.guifontname = name
	}
}
