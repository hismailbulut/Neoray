package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/adrg/sysfont"
	rl "github.com/chunqian/go-raylib/raylib"
)

type Font struct {
	size float32

	regular_found     bool
	italic_found      bool
	bold_found        bool
	bold_italic_found bool

	regular     rl.Font
	italic      rl.Font
	bold        rl.Font
	bold_italic rl.Font

	system_default_fontname string
}

func (font *Font) Load(fontname string, size float32) {

	if size < 5 {
		size = 12
	}

	font.size = size

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

	font.find_and_load(fontname)
}

func (font *Font) Unload() {
	rl.UnloadFont(font.regular)
	rl.UnloadFont(font.bold)
	rl.UnloadFont(font.italic)
	rl.UnloadFont(font.bold_italic)
}

func (font *Font) GetDrawableFont(italic bool, bold bool) rl.Font {
	if italic && bold {
		return font.bold_italic
	} else if italic && !bold {
		return font.italic
	} else if bold && !italic {
		return font.bold
	}
	return font.regular
}

func (font *Font) find_and_load(fontname string) {

	finder := sysfont.NewFinder(nil)
	font_list := finder.List()

	matched_fonts, ok := font.get_matching_fonts(fontname, font_list)
	if !ok {
		fmt.Println("Font", fontname, "not found. Using system default font.")
		matched_fonts, _ = font.get_matching_fonts(font.system_default_fontname, font_list)
	}

	if !font.load_matching_fonts(matched_fonts, "Light", "Extra", "Semi", "Medium") {
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
			font.bold_italic = rl.LoadFontEx(f.Filename, int32(font.size), nil, 4096)
			font.bold_italic_found = true
			fmt.Println("FONT: Bold Italic:", f.Filename)
		} else if has_italic && !has_bold && !font.italic_found {
			font.italic = rl.LoadFontEx(f.Filename, int32(font.size), nil, 4096)
			font.italic_found = true
			fmt.Println("FONT: Italic:", f.Filename)
		} else if has_bold && !has_italic && !font.bold_found {
			font.bold = rl.LoadFontEx(f.Filename, int32(font.size), nil, 4096)
			font.bold_found = true
			fmt.Println("FONT: Bold:", f.Filename)
		} else if !has_bold && !has_italic && !font.regular_found {
			font.regular = rl.LoadFontEx(f.Filename, int32(font.size), nil, 4096)
			font.regular_found = true
			fmt.Println("FONT: Regular:", f.Filename)
		}
	}

	return font.regular_found && font.italic_found && font.bold_found && font.bold_italic_found
}

func (font *Font) contains(f *sysfont.Font, str string) bool {
	return strings.Contains(f.Name, str) ||
		strings.Contains(f.Family, str) ||
		strings.Contains(f.Filename, str)
}
