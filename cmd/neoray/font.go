package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/adrg/sysfont"
	"github.com/veandco/go-sdl2/ttf"
)

type Font struct {
	size float32

	regular_found     bool
	italic_found      bool
	bold_found        bool
	bold_italic_found bool

	regular     *ttf.Font
	italic      *ttf.Font
	bold        *ttf.Font
	bold_italic *ttf.Font

	system_default_fontname string
}

func CreateFont(fontname string, size float32) Font {

	if err := ttf.Init(); err != nil {
		log.Fatalln(err)
	}

	if size < 6 {
		size = 12
	}
	font := Font{size: size}

	switch runtime.GOOS {
	case "windows":
		font.system_default_fontname = "Consolas"
		break
	case "linux":
		font.system_default_fontname = "Noto Sans Mono"
		break
	case "darwin":
		font.system_default_fontname = "Menlo"
		break
	}

	if fontname == "" {
		font.find_and_load(font.system_default_fontname)
	} else {
		font.find_and_load(fontname)
	}
	return font
}

func (font *Font) Unload() {
	font.regular.Close()
	font.bold.Close()
	font.italic.Close()
	font.bold_italic.Close()
	ttf.Quit()
}

func (font *Font) GetDrawableFont(italic bool, bold bool) *ttf.Font {
	if italic && bold {
		return font.bold_italic
	} else if italic && !bold {
		return font.italic
	} else if bold && !italic {
		return font.bold
	}
	return font.regular
}

func (font *Font) GetCellSize() (int, int) {
	metrics, err := font.regular.GlyphMetrics('m')
	if err != nil {
		fmt.Println(err)
		return int(font.size), int(font.size / 2)
	}
	w := metrics.Advance
	h := font.regular.Height() + metrics.MinY
	return w, h
}

func (font *Font) find_and_load(fontname string) {

	finder := sysfont.NewFinder(nil)
	font_list := finder.List()

	matched_fonts, ok := font.get_matching_fonts(fontname, font_list)
	if !ok {
		fmt.Println("Font", fontname, "not found. Using system default font.")
		matched_fonts, _ = font.get_matching_fonts(font.system_default_fontname, font_list)
	}

	if !font.load_matching_fonts(matched_fonts, "Light", "Extra") {
		matched_fonts, _ = font.get_matching_fonts(font.system_default_fontname, font_list)
		font.load_matching_fonts(matched_fonts)
	}
}

func (font *Font) get_matching_fonts(name string, list []*sysfont.Font) ([]sysfont.Font, bool) {
	matched_fonts := []sysfont.Font{}
	for _, f := range list {
		if font.contains(f, name) {
			matched_fonts = append(matched_fonts, *f)
		}
	}
	return matched_fonts, len(matched_fonts) > 0
}

func (font *Font) load_matching_fonts(font_list []sysfont.Font, ignore_words ...string) bool {

	for _, f := range font_list {

		has_italic := font.contains(&f, "Italic")
		has_bold := font.contains(&f, "Bold")

		for _, w := range ignore_words {
			if font.contains(&f, w) {
				continue
			}
		}

		if has_italic && has_bold && !font.bold_italic_found {
			font.bold_italic = font.load_font_data(f.Filename)
			font.bold_italic_found = true
		} else if has_italic && !has_bold && !font.italic_found {
			font.italic = font.load_font_data(f.Filename)
			font.italic_found = true
		} else if has_bold && !has_italic && !font.bold_found {
			font.bold = font.load_font_data(f.Filename)
			font.bold_found = true
		} else if !has_bold && !has_italic && !font.regular_found {
			font.regular = font.load_font_data(f.Filename)
			font.regular_found = true
		}
	}

	return font.regular_found && font.italic_found && font.bold_found && font.bold_italic_found
}

func (font *Font) load_font_data(filename string) *ttf.Font {
	font_data, err := ttf.OpenFont(filename, int(font.size))
	if err != nil {
		fmt.Println(err)
	}
	return font_data
}

func (font *Font) contains(f *sysfont.Font, str string) bool {
	return strings.Contains(f.Name, str) ||
		strings.Contains(f.Family, str) ||
		strings.Contains(f.Filename, str)
}
