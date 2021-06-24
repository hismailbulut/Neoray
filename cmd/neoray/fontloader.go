package main

import (
	_ "embed"
	"strings"

	"github.com/adrg/sysfont"
)

var (
	//go:embed fonts/CascadiaMono-Regular.ttf
	defaultRegularData []byte
	//go:embed fonts/CascadiaMono-BoldItalic.otf
	defaultBoldItalicData []byte
	//go:embed fonts/CascadiaMono-Italic.otf
	defaultItalicData []byte
	//go:embed fonts/CascadiaMono-Bold.otf
	defaultBoldData []byte
	// List of installed system fonts.
	systemFontList []*sysfont.Font = nil
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

func CheckSystemFonts() {
	// On windows systems this takes long (2-3 secs)
	// We could do it on beginning in another goroutine
	defer measure_execution_time("CheckSystemFonts")()
	if systemFontList == nil {
		systemFontList = sysfont.NewFinder(nil).List()
	}
}

func CreateFont(fontName string, size float32) (Font, bool) {
	defer measure_execution_time("CreateFont")()
	CheckSystemFonts()

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
	font.bold_italic.Resize(newsize)
	font.italic.Resize(newsize)
	font.bold.Resize(newsize)
	font.regular.Resize(newsize)
}

func (font *Font) GetSuitableFace(italic bool, bold bool) *FontFace {
	if font.bold_italic.loaded && italic && bold {
		return &font.bold_italic
	} else if font.italic.loaded && italic {
		return &font.italic
	} else if font.bold.loaded && bold {
		return &font.bold
	}
	return &font.regular
}

func (font *Font) GetCellSize() (int, int) {
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

	var name string
	var ok bool
	// bold-italic
	name, ok = loadBestMatch(&font.bold_italic, bold_italics, font.size)
	if ok {
		log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Bold Italic Face:", name)
	}
	// italic
	name, ok = loadBestMatch(&font.italic, italics, font.size)
	if ok {
		log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Italic Face:", name)
	}
	//bold
	name, ok = loadBestMatch(&font.bold, bolds, font.size)
	if ok {
		log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Bold Face:", name)
	}
	//regular
	name, ok = loadBestMatch(&font.regular, others, font.size)
	if ok {
		log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Regular Face:", name)
	}

	return font.regular.loaded
}

func loadBestMatch(face *FontFace, list []sysfont.Font, size float32) (string, bool) {
	if len(list) > 0 && !face.loaded {
		smallest := findSmallestFontName(list)
		var err error
		*face, err = CreateFontFace(smallest, size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, err, "in CreateFontFace")
			return "", false
		}
		return smallest, true
	}
	return "", false
}

// Finding smallest length is very bad idea, but works on ~half of the fonts.
// The best we could do is native font enumerating. But I don't know if it's
// possible to get font data or path from it's family name.
func findSmallestFontName(font_list []sysfont.Font) string {
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
