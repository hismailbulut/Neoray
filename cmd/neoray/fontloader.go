package main

import (
	_ "embed"
	"strings"

	"github.com/adrg/sysfont"
)

var (
	//go:embed assets/CascadiaMono-Regular.ttf
	defaultRegularData []byte
	//go:embed assets/CascadiaMono-BoldItalic.otf
	defaultBoldItalicData []byte
	//go:embed assets/CascadiaMono-Italic.otf
	defaultItalicData []byte
	//go:embed assets/CascadiaMono-Bold.otf
	defaultBoldData []byte

	systemFontList []*sysfont.Font
)

type Font struct {
	// If you want to disable a font, just set size to 0.
	size        float32
	regular     FontFace
	italic      FontFace
	bold        FontFace
	bold_italic FontFace
}

func CreateDefaultFont() Font {
	defer measure_execution_time("InitializeFontLoader")()
	systemFontList = sysfont.NewFinder(nil).List()
	// Load the default font.
	font := Font{
		size: DEFAULT_FONT_SIZE,
	}
	var check = func(err error) {
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, err)
			log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to load default font! Shutting down.")
		}
	}
	var err error
	// regular
	regular, err := CreateFontFaceMemory(defaultRegularData, font.size)
	check(err)
	font.regular = regular
	// bold italic
	bold_italic, err := CreateFontFaceMemory(defaultBoldItalicData, font.size)
	check(err)
	font.bold_italic = bold_italic
	// italic
	italic, err := CreateFontFaceMemory(defaultItalicData, font.size)
	check(err)
	font.italic = italic
	// bold
	bold, err := CreateFontFaceMemory(defaultBoldData, font.size)
	check(err)
	font.bold = bold
	return font
}

func CreateFont(fontName string, size float32) (Font, bool) {
	defer measure_execution_time("CreateFont")()

	if fontName == "" || fontName == " " {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Font name can not be empty!")
		return Font{}, false
	}

	if size < MINIMUM_FONT_SIZE {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY,
			"Font size", size, "is small. Reset to default", DEFAULT_FONT_SIZE)
		size = DEFAULT_FONT_SIZE
	}

	font := Font{size: size}
	if !font.findAndLoad(fontName) {
		return font, false
	}

	return font, true
}

func (font *Font) Resize(newsize float32) {
	if newsize == font.size {
		return
	}
	font.size = newsize
	if font.bold_italic.size > 0 {
		if err := font.bold_italic.Resize(newsize); err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, err)
		}
	}
	if font.italic.size > 0 {
		if err := font.italic.Resize(newsize); err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, err)
		}
	}
	if font.bold.size > 0 {
		if err := font.bold.Resize(newsize); err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, err)
		}
	}
	if font.regular.size > 0 {
		if err := font.regular.Resize(newsize); err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, err)
		}
	}
}

func (font *Font) GetSuitableFace(italic bool, bold bool) *FontFace {
	if italic && bold && font.bold_italic.size > 0 {
		return &font.bold_italic
	} else if italic && font.italic.size > 0 {
		return &font.italic
	} else if bold && font.bold.size > 0 {
		return &font.bold
	}
	return &font.regular
}

func (font *Font) CalculateCellSize() (int, int) {
	return font.regular.advance, font.regular.height
}

func (font *Font) findAndLoad(fontName string) bool {
	matched_fonts, ok := font.getMatchingFonts(fontName)
	if !ok || !font.loadMatchingFonts(matched_fonts) {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Font", fontName, "not found.")
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
	if font.bold_italic.size == 0 && len(bold_italics) > 0 {
		smallest := findSmallestLengthFont(bold_italics)
		font.bold_italic, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load bold italic font face.")
		} else {
			log_debug("Font bold italic:", smallest)
		}
	}

	// italic
	if font.italic.size == 0 && len(italics) > 0 {
		smallest := findSmallestLengthFont(italics)
		font.italic, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load italic font face.")
		} else {
			log_debug("Font italic:", smallest)
		}
	}

	//bold
	if font.bold.size == 0 && len(bolds) > 0 {
		smallest := findSmallestLengthFont(bolds)
		font.bold, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load bold font face.")
		} else {
			log_debug("Font bold:", smallest)
		}
	}

	//regular
	if font.regular.size == 0 && len(others) > 0 {
		smallest := findSmallestLengthFont(others)
		font.regular, err = CreateFontFace(smallest, font.size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load regular font face.")
			return false
		} else {
			log_debug("Font regular:", smallest)
		}
	}

	return font.regular.size > 0
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
