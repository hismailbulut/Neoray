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
		// if this is the system default font, we already loaded it
		if name == systemFontDefault {
			log_debug_msg("Already loaded system font.")
			// Unload current font. Because user wants system default.
			EditorSingleton.renderer.SetDefaultFont(size)
			return
		}
		// Create and set renderers font.
		log_debug_msg("Loading font:", name)
		font, ok := CreateFont(name, size)
		if !ok {
			return
		}
		EditorSingleton.renderer.SetFont(font)
	}
}
