package main

import (
	"path/filepath"

	"github.com/hismailbulut/neoray/src/fontfinder"
	"github.com/hismailbulut/neoray/src/hacknerd"
)

type Font struct {
	// If you want to disable a font, just set size to 0.
	size        float32
	regular     FontFace
	bold_italic FontFace
	italic      FontFace
	bold        FontFace
}

func CreateDefaultFont() Font {
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
	regular, err := CreateFaceFromMem(hacknerd.Regular, font.size)
	check(err)
	font.regular = regular
	// bold italic
	bold_italic, err := CreateFaceFromMem(hacknerd.BoldItalic, font.size)
	check(err)
	font.bold_italic = bold_italic
	// italic
	italic, err := CreateFaceFromMem(hacknerd.Italic, font.size)
	check(err)
	font.italic = italic
	// bold
	bold, err := CreateFaceFromMem(hacknerd.Bold, font.size)
	check(err)
	font.bold = bold
	return font
}

func CreateFont(fontName string, size float32) (Font, bool) {
	defer measure_execution_time()()

	assert(fontName != "", "Font name can not be empty!")

	if size < MINIMUM_FONT_SIZE {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY,
			"Font size", size, "is small and set to default", DEFAULT_FONT_SIZE)
		size = DEFAULT_FONT_SIZE
	}

	font := Font{size: size}

	info := fontfinder.Find(fontName)

	var err error
	if info.Regular != "" {
		font.regular, err = CreateFace(info.Regular, size)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to load regular font.", err)
			return font, false
		} else {
			log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Regular:", filepath.Base(info.Regular))
		}
	} else {
		return font, false
	}

	if info.BoldItalic != "" {
		font.bold_italic, err = CreateFace(info.BoldItalic, size)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Failed to load bold italic font.", err)
		} else {
			log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Bold Italic:", filepath.Base(info.BoldItalic))
		}
	}

	if info.Italic != "" {
		font.italic, err = CreateFace(info.Italic, size)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Failed to load italic font.", err)
		} else {
			log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Italic:", filepath.Base(info.Italic))
		}
	}

	if info.Bold != "" {
		font.bold, err = CreateFace(info.Bold, size)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Failed to load bold font.", err)
		} else {
			log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Bold:", filepath.Base(info.Bold))
		}
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
	// TODO: Return nil.
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
