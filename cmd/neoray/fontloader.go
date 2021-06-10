package main

import (
	"runtime"
	"strings"

	"github.com/adrg/sysfont"
)

var (
	systemFontDefault string
	systemFontList    []*sysfont.Font
)

type Font struct {
	// If you want to disable a font, just set size to 0.
	size int
	// All faces has a bool value that specifies is font loaded or not.
	// And only set to true by CreateFontFace
	regular     FontFace
	italic      FontFace
	bold        FontFace
	bold_italic FontFace
}

func FontSystemInit() {
	systemFontList = sysfont.NewFinder(nil).List()
	switch runtime.GOOS {
	case "windows":
		systemFontDefault = "Consolas"
		break
	case "linux":
		systemFontDefault = "Noto Sans Mono"
		break
	case "darwin":
		systemFontDefault = "Menlo"
		break
	}
}

func CreateFont(fontName string, size int) (Font, bool) {
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

func (font *Font) Unload() {
	font.bold_italic.loaded = false
	font.bold_italic.handle = nil
	font.italic.loaded = false
	font.italic.handle = nil
	font.bold.loaded = false
	font.bold.handle = nil
	font.regular.loaded = false
	font.regular.handle = nil
	font.size = 0
}

func (font *Font) GetSuitableFont(italic bool, bold bool) FontFace {
	if italic && bold && font.bold_italic.loaded {
		return font.bold_italic
	} else if italic && font.italic.loaded {
		return font.italic
	} else if bold && font.bold.loaded {
		return font.bold
	}
	return font.regular
}

func (font *Font) CalculateCellSize() (int, int) {
	return font.regular.advance, font.regular.ascent + font.regular.descent
}

func (font *Font) loadDefaultFont() {
	matched_fonts, ok := font.getMatchingFonts(systemFontDefault)
	if !ok || !font.loadMatchingFonts(matched_fonts) {
		// Maybe default system font is not installed (?) or failed to access.
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY,
			"Default system font is not found!", systemFontDefault)
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

	var err error

	// bold-italic
	if !font.bold_italic.loaded && len(bold_italics) > 0 {
		smallest := findSmallestLengthFont(bold_italics)
		font.bold_italic, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load bold italic font face.")
		}
	}

	// italic
	if !font.italic.loaded && len(italics) > 0 {
		smallest := findSmallestLengthFont(italics)
		font.italic, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load italic font face.")
		}
	}

	//bold
	if !font.bold.loaded && len(bolds) > 0 {
		smallest := findSmallestLengthFont(bolds)
		font.bold, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load bold font face.")
		}
	}

	//regular
	if !font.regular.loaded && len(others) > 0 {
		smallest := findSmallestLengthFont(others)
		font.regular, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load regular font face.")
			return false
		}
	}

	return true
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
