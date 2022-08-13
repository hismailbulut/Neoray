package main

import (
	"strconv"
	"strings"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/fontkit"
	"github.com/hismailbulut/Neoray/pkg/logger"
)

const DEFAULT_FONT_SIZE = 12

// neovim ui options
type UIOptions struct {
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

func CreateUIOptions() UIOptions {
	return UIOptions{
		mousehide: true,
	}
}

func (options *UIOptions) setGuiFont(guifont string) {
	// Load Font
	if guifont == options.guifont {
		return
	}
	options.guifont = guifont
	var size float64 = DEFAULT_FONT_SIZE
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
				size = tsize
			}
		}
	}
	if name == "" {
		// Set nil to disable font
		Editor.gridManager.SetGridFontKit(1, nil)
		Editor.contextMenu.SetFontKit(nil)
	} else {
		// Create and set font
		logger.Log(logger.TRACE, "Loading font", name)
		kit, err := fontkit.CreateKit(name)
		if err != nil {
			Editor.nvim.EchoError("Font %s not found", name)
		} else {
			// Log some info
			if kit.Regular() != nil {
				logger.Log(logger.TRACE, "Regular:", kit.Regular().FilePath())
			}
			if kit.Bold() != nil {
				logger.Log(logger.TRACE, "Bold:", kit.Bold().FilePath())
			}
			if kit.Italic() != nil {
				logger.Log(logger.TRACE, "Italic:", kit.Italic().FilePath())
			}
			if kit.BoldItalic() != nil {
				logger.Log(logger.TRACE, "BoldItalic:", kit.BoldItalic().FilePath())
			}
			// Set fonts
			Editor.gridManager.SetGridFontKit(1, kit)
			Editor.contextMenu.SetFontKit(kit)
		}
	}
	// Always set font size to default if user not set
	Editor.gridManager.SetGridFontSize(1, size)
	Editor.contextMenu.SetFontSize(size)
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
