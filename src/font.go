package main

import (
	"path/filepath"

	"github.com/hismailbulut/neoray/src/caskaydia"
	"github.com/hismailbulut/neoray/src/fontfinder"
)

const (
	MINIMUM_FONT_SIZE = 5
	DEFAULT_FONT_SIZE = 11
)

// If you want to disable a font, just set size to 0.
type Font struct {
	size        float32
	name        string
	regular     *FontFace
	bold_italic *FontFace
	italic      *FontFace
	bold        *FontFace
}

func CreateDefaultFont() Font {
	defer measure_execution_time()()

	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Loading default font.")

	font := Font{
		size: DEFAULT_FONT_SIZE,
	}
	var check = func(err error) {
		if err != nil {
			logMessage(LEVEL_ERROR, TYPE_NEORAY, err)
			logMessage(LEVEL_FATAL, TYPE_NEORAY, "Failed to load default font!")
		}
	}
	var err error
	// regular
	regular, err := CreateFaceFromMem(caskaydia.Regular, font.size)
	check(err)
	font.regular = regular
	font.name = "Default"
	// bold italic
	bold_italic, err := CreateFaceFromMem(caskaydia.BoldItalic, font.size)
	check(err)
	font.bold_italic = bold_italic
	// italic
	italic, err := CreateFaceFromMem(caskaydia.Italic, font.size)
	check(err)
	font.italic = italic
	// bold
	bold, err := CreateFaceFromMem(caskaydia.Bold, font.size)
	check(err)
	font.bold = bold

	// TODO: Do we need italic and bold for default font?
	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Default font loaded.")

	return font
}

func CreateFont(fontName string, size float32) (Font, bool) {
	defer measure_execution_time()()

	logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Loading font", fontName, "with size", size)

	if size < MINIMUM_FONT_SIZE {
		logMessage(LEVEL_WARN, TYPE_NEORAY,
			"Font size", size, "is small and automatically set to default", DEFAULT_FONT_SIZE)
		size = DEFAULT_FONT_SIZE
	}

	font := Font{size: size}

	info := fontfinder.Find(fontName)

	var err error
	if info.Regular != "" {
		font.regular, err = CreateFace(info.Regular, size)
		if err != nil {
			logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to load regular font.", err)
			return font, false
		} else {
			logMessage(LEVEL_TRACE, TYPE_NEORAY, "Regular:", filepath.Base(info.Regular))
			font.name = font.regular.FamilyName()
			if font.name == "" {
				font.name = "Unknown Family Name"
			}
		}
	} else {
		return font, false
	}

	if info.BoldItalic != "" {
		font.bold_italic, err = CreateFace(info.BoldItalic, size)
		if err != nil {
			logMessage(LEVEL_WARN, TYPE_NEORAY, "Failed to load bold italic font.", err)
		} else {
			logMessage(LEVEL_TRACE, TYPE_NEORAY, "Bold Italic:", filepath.Base(info.BoldItalic))
		}
	} else {
		logMessage(LEVEL_WARN, TYPE_NEORAY, "Font has no bold italic face.")
	}

	if info.Italic != "" {
		font.italic, err = CreateFace(info.Italic, size)
		if err != nil {
			logMessage(LEVEL_WARN, TYPE_NEORAY, "Failed to load italic font.", err)
		} else {
			logMessage(LEVEL_TRACE, TYPE_NEORAY, "Italic:", filepath.Base(info.Italic))
		}
	} else {
		logMessage(LEVEL_WARN, TYPE_NEORAY, "Font has no italic face.")
	}

	if info.Bold != "" {
		font.bold, err = CreateFace(info.Bold, size)
		if err != nil {
			logMessage(LEVEL_WARN, TYPE_NEORAY, "Failed to load bold font.", err)
		} else {
			logMessage(LEVEL_TRACE, TYPE_NEORAY, "Bold:", filepath.Base(info.Bold))
		}
	} else {
		logMessage(LEVEL_WARN, TYPE_NEORAY, "Font has no bold face.")
	}

	return font, true
}

func (font *Font) Resize(newsize float32) {
	if newsize < MINIMUM_FONT_SIZE {
		newsize = MINIMUM_FONT_SIZE
	}
	// Regular face is always non nil, but others may be
	assert(font.regular != nil, "Font's regular face can not be nil.")
	font.regular.Resize(newsize)
	if font.bold_italic != nil {
		font.bold_italic.Resize(newsize)
	}
	if font.italic != nil {
		font.italic.Resize(newsize)
	}
	if font.bold != nil {
		font.bold.Resize(newsize)
	}
	font.size = newsize
}

// This function returns nil when there is no requested font style
func (font *Font) GetSuitableFace(italic bool, bold bool) *FontFace {
	if italic && bold {
		return font.bold_italic
	} else if italic {
		return font.italic
	} else if bold {
		return font.bold
	}
	return font.regular
}

func (font *Font) GetCellSize() (int, int) {
	return font.regular.advance, font.regular.height
}
