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
	// our options
	smoothcursor bool
}

func (options *UIOptions) SetGuiFont(font string) {
	// Load Font
	if font != "" && font != " " && font != options.guifont {
		options.setUserFont(font)
		options.guifont = font
	}
}

func (options *UIOptions) setUserFont(guifont string) {
	options.guifont = guifont
	size := DEFAULT_FONT_SIZE
	// treat underlines like whitespaces
	guifont = strings.ReplaceAll(guifont, "_", " ")
	// parse font options
	fontOptions := strings.Split(guifont, ":")
	name := fontOptions[0]
	for _, opt := range fontOptions[1:] {
		// TODO:h guifont
		if opt[0] == 'h' {
			tsize, err := strconv.Atoi(opt[1:])
			if err == nil {
				size = tsize
			}
		}
	}
	// if this is the system default font, we already loaded it
	if name == system_default_fontname {
		log_debug_msg("Already loaded system font.")
		// Unload current font
		EditorSingleton.renderer.SetFont(Font{})
		return
	}
	// create and set renderers font
	log_debug_msg("Loading font:", name)
	font, ok := CreateFont(name, float32(size))
	if !ok {
		return
	}
	EditorSingleton.renderer.SetFont(font)
}
