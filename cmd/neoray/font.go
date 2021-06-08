package main

import (
	"runtime"
	"strings"

	"github.com/adrg/sysfont"
	"github.com/veandco/go-sdl2/ttf"
)

var (
	system_default_fontname string
	systemFontList          []*sysfont.Font
)

type Font struct {
	size        float32
	regular     *ttf.Font
	italic      *ttf.Font
	bold        *ttf.Font
	bold_italic *ttf.Font
}

func FontSystemInit() {
	if err := ttf.Init(); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL_TTF:", err)
	}
	systemFontList = sysfont.NewFinder(nil).List()
	switch runtime.GOOS {
	case "windows":
		system_default_fontname = "Consolas"
		break
	case "linux":
		system_default_fontname = "Noto Sans Mono"
		break
	case "darwin":
		system_default_fontname = "Menlo"
		break
	}
}

func FontSystemClose() {
	ttf.Quit()
}

func CreateFont(fontName string, size float32) (Font, bool) {
	defer measure_execution_time("CreateFont")()

	if size < MINIMUM_FONT_SIZE {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY,
			"Font size", size, "is small. Reset to default", DEFAULT_FONT_SIZE)
		size = DEFAULT_FONT_SIZE
	}

	font := Font{size: size}
	if fontName == "" || fontName == " " {
		font.loadDefaultFont()
	} else if !font.findAndLoad(fontName) {
		return font, false
	}

	return font, true
}

func (font *Font) GetSuitableFont(italic bool, bold bool) *ttf.Font {
	if italic && bold {
		if font.bold_italic == nil {
			font.regular.SetStyle(ttf.STYLE_BOLD | ttf.STYLE_ITALIC)
		} else {
			return font.bold_italic
		}
	} else if italic {
		if font.italic == nil {
			font.regular.SetStyle(ttf.STYLE_ITALIC)
		} else {
			return font.italic
		}
	} else if bold {
		if font.bold == nil {
			font.regular.SetStyle(ttf.STYLE_BOLD)
		} else {
			return font.bold
		}
	} else {
		font.regular.SetStyle(0)
	}
	return font.regular
}

func (font *Font) CalculateCellSize() (int, int) {
	metrics, err := font.regular.GlyphMetrics('M')
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to calculate cell size:", err)
		return int(font.size / 2), int(font.size)
	}
	w := metrics.Advance
	h := font.regular.Height()
	return w, h
}

func (font *Font) loadDefaultFont() {
	matched_fonts, ok := font.getMatchingFonts(system_default_fontname)
	if !ok || !font.loadMatchingFonts(matched_fonts) {
		// Maybe default system font is not installed (?) or failed to access.
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY,
			"Default system font is not found!", system_default_fontname)
	}
}

func (font *Font) findAndLoad(fontName string) bool {
	matched_fonts, ok := font.getMatchingFonts(fontName)
	if !ok || !font.loadMatchingFonts(matched_fonts) {
		// Maybe non regular fonts are loaded?
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Font", fontName, "not found.")
		font.Unload()
		return false
	}
	return true
}

func (font *Font) getMatchingFonts(fontName string) ([]sysfont.Font, bool) {
	matched_fonts := []sysfont.Font{}
	for _, f := range systemFontList {
		if fontNameContains(f, fontName) {
			matched_fonts = append(matched_fonts, *f)
		}
	}
	return matched_fonts, len(matched_fonts) > 0
}

func (font *Font) loadMatchingFonts(font_list []sysfont.Font) bool {
	bold_italics := make([]sysfont.Font, 0)
	italics := make([]sysfont.Font, 0)
	bolds := make([]sysfont.Font, 0)
	others := make([]sysfont.Font, 0)

	for _, f := range font_list {
		has_italic := fontNameContains(&f, "Italic")
		has_bold := fontNameContains(&f, "Bold")
		if has_italic && has_bold {
			bold_italics = append(bold_italics, f)
		} else if has_italic && !has_bold {
			italics = append(italics, f)
		} else if has_bold && !has_italic {
			bolds = append(bolds, f)
		} else if !has_bold && !has_italic {
			others = append(others, f)
		}
	}

	// bold-italic
	if font.bold_italic == nil && len(bold_italics) > 0 {
		smallest := findSmallestLengthFont(bold_italics)
		font.bold_italic = font.loadFontData(smallest)
		if font.bold_italic != nil {
			log_debug_msg("Font Bold Italic:", font.bold_italic.FaceFamilyName(), "Bold Italic")
		}
	}

	// italic
	if font.italic == nil && len(italics) > 0 {
		smallest := findSmallestLengthFont(italics)
		font.italic = font.loadFontData(smallest)
		if font.italic != nil {
			log_debug_msg("Font Italic:", font.italic.FaceFamilyName(), "Italic")
		}
	}

	//bold
	if font.bold == nil && len(bolds) > 0 {
		smallest := findSmallestLengthFont(bolds)
		font.bold = font.loadFontData(smallest)
		if font.bold != nil {
			log_debug_msg("Font Bold:", font.bold.FaceFamilyName(), "Bold")
		}
	}

	//regular
	if font.regular == nil && len(others) > 0 {
		smallest := findSmallestLengthFont(others)
		font.regular = font.loadFontData(smallest)
		if font.regular != nil {
			log_debug_msg("Font Regular:", font.regular.FaceFamilyName())
		} else {
			return false
		}
	}

	return true
}

func (font *Font) loadFontData(filename string) *ttf.Font {
	sdl_font_data, err := ttf.OpenFont(filename, int(font.size))
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to open font file:", err)
		return nil
	}
	return sdl_font_data
}

func findSmallestLengthFont(font_list []sysfont.Font) string {
	smallest := ""
	smallestLen := 1000000
	for _, f := range font_list {
		if len(f.Filename) < smallestLen {
			smallest = f.Filename
			smallestLen = len(f.Filename)
		}
	}
	return smallest
}

func fontNameContains(f *sysfont.Font, str string) bool {
	return strings.Contains(strings.ToLower(f.Name), strings.ToLower(str)) ||
		strings.Contains(strings.ToLower(f.Family), strings.ToLower(str)) ||
		strings.Contains(strings.ToLower(f.Filename), strings.ToLower(str))
}

func (font *Font) Unload() {
	if font.regular != nil {
		font.regular.Close()
		font.regular = nil
	}
	if font.bold != nil {
		font.bold.Close()
		font.bold = nil
	}
	if font.italic != nil {
		font.italic.Close()
		font.italic = nil
	}
	if font.bold_italic != nil {
		font.bold_italic.Close()
		font.bold_italic = nil
	}
	font.size = 0
}
