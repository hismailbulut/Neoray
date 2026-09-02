package main

import (
	"strconv"
	"strings"

	"github.com/hismailbulut/Neoray/pkg/common"
)

const DEFAULT_FONT_SIZE = 12

// neovim ui options
type UIOptions struct {
	editor        *Editor
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
	mousehide     bool // will be implemented soon, currently always true
}

func CreateUIOptions(editor *Editor) UIOptions {
	return UIOptions{
		editor:    editor,
		mousehide: true,
	}
}

func (options *UIOptions) setGuiFont(guifont string) {
	// Load Font
	if guifont == options.guifont {
		return
	}
	options.guifont = guifont
	var fontSize float64 = DEFAULT_FONT_SIZE
	// comma seperates font names
	fonts := strings.Split(guifont, ",")
	fontNames := make([]string, 0)
	for i := len(fonts) - 1; i >= 0; i-- {
		font := fonts[i]
		// treat underlines like whitespaces ?(maybe we shouln't)
		font = strings.ReplaceAll(font, "_", " ")
		font = strings.TrimSpace(font)
		// parse font options
		fontOptions := strings.Split(font, ":")
		fontName := fontOptions[0]
		for _, opt := range fontOptions[1:] {
			// TODO: there are different font options than 'h'
			if len(opt) > 1 && opt[0] == 'h' {
				// Font size
				_fontSize, err := strconv.ParseFloat(opt[1:], 32)
				if err == nil {
					fontSize = _fontSize
				}
			}
		}
		if strings.TrimSpace(fontName) != "" {
			fontNames = append(fontNames, fontName)
		}
	}
	options.editor.fontManager.SetFontSize(fontSize)
	options.editor.fontManager.SetFontNames(fontNames)
}

type HighlightAttribute struct {
	foreground    common.Color
	background    common.Color
	special       common.Color
	reverse       bool
	italic        bool
	bold          bool
	strikethrough bool
	underline     bool
	// underlineline bool
	undercurl bool
	// underdot  bool
	// underdash bool
	// blend     int
	// TODO: Implement commented attributes
}

type ModeInfo struct {
	cursor_shape    string
	cell_percentage int
	blinkwait       int
	blinkon         int
	blinkoff        int
	attr_id         int
	attr_id_lm      int
	short_name      string
	name            string
}

type Mode struct {
	cursor_style_enabled bool
	mode_infos           []ModeInfo
	current_mode_name    string
	current_mode         int
}

func (mode *Mode) Current() ModeInfo {
	if mode.current_mode < len(mode.mode_infos) {
		return mode.mode_infos[mode.current_mode]
	}
	return ModeInfo{}
}

func (mode *Mode) Clear() {
	mode.mode_infos = []ModeInfo{}
}

func (mode *Mode) Add(info ModeInfo) {
	mode.mode_infos = append(mode.mode_infos, info)
}
