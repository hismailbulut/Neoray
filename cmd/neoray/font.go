package main

import (
	"runtime"
	"strings"

	"github.com/adrg/sysfont"
	"github.com/veandco/go-sdl2/ttf"
	"golang.org/x/image/font/opentype"
)

const MAX_CHARS = 4096

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
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL_TTF:", err)
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

	// italic text
	print_font_information(font.regular)

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

func (font *Font) CalculateCellSize() (int, int) {
	metrics, err := font.regular.GlyphMetrics('m')
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
		return int(font.size), int(font.size / 2)
	}
	w := metrics.Advance
	h := font.regular.Height()
	return w, h
}

func (font *Font) find_and_load(fontname string) {
	finder := sysfont.NewFinder(nil)
	font_list := finder.List()
	matched_fonts, ok := font.get_matching_fonts(fontname, font_list)
	if !ok {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Font", fontname, "not found. Using system default font.")
		matched_fonts, _ = font.get_matching_fonts(font.system_default_fontname, font_list)
	}
	if !font.load_matching_fonts(matched_fonts) {
		matched_fonts, _ = font.get_matching_fonts(font.system_default_fontname, font_list)
		font.load_matching_fonts(matched_fonts)
	}
}

func (font *Font) get_matching_fonts(name string, list []*sysfont.Font) ([]sysfont.Font, bool) {
	matched_fonts := []sysfont.Font{}
	for _, f := range list {
		if font_name_contains(f, name) {
			matched_fonts = append(matched_fonts, *f)
		}
	}
	return matched_fonts, len(matched_fonts) > 0
}

func (font *Font) load_matching_fonts(font_list []sysfont.Font) bool {

	bold_italics := make([]sysfont.Font, 0)
	italics := make([]sysfont.Font, 0)
	bolds := make([]sysfont.Font, 0)
	others := make([]sysfont.Font, 0)

	for _, f := range font_list {
		has_italic := font_name_contains(&f, "Italic")
		has_bold := font_name_contains(&f, "Bold")
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
	if !font.bold_italic_found && len(bold_italics) > 0 {
		bold_italic_font_file_name := find_smaller_length_font_name(bold_italics)
		font.bold_italic = font.load_font_data(bold_italic_font_file_name)
		if font.bold_italic != nil {
			font.bold_italic_found = true
			log_debug_msg("Font Bold Italic:", bold_italic_font_file_name)
		}
	}

	// italic
	if !font.italic_found && len(italics) > 0 {
		italic_font_file_name := find_smaller_length_font_name(italics)
		font.italic = font.load_font_data(italic_font_file_name)
		if font.italic != nil {
			font.italic_found = true
			log_debug_msg("Font Italic:", italic_font_file_name)
		}
	}

	//bold
	if !font.bold_found && len(bolds) > 0 {
		bold_font_file_name := find_smaller_length_font_name(bolds)
		font.bold = font.load_font_data(bold_font_file_name)
		if font.bold != nil {
			font.bold_found = true
			log_debug_msg("Font Bold:", bold_font_file_name)
		}
	}

	//regular
	if !font.regular_found && len(others) > 0 {
		regular_font_file_name := find_smaller_length_font_name(others)
		font.regular = font.load_font_data(regular_font_file_name)
		if font.regular != nil {
			font.regular_found = true
			log_debug_msg("Font Regular:", regular_font_file_name)
		}
	}

	return font.regular_found && font.bold_found && font.italic_found && font.bold_italic_found
}

func (font *Font) load_font_data(filename string) *ttf.Font {
	font_data, err := ttf.OpenFont(filename, int(font.size))
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to open font file:", err)
		return nil
	}
	return font_data
}

func find_smaller_length_font_name(font_list []sysfont.Font) string {
	best_match_font_file_name := ""
	smallest_font_name_length := 1000000
	for _, f := range font_list {
		if len(f.Filename) < smallest_font_name_length {
			best_match_font_file_name = f.Filename
			smallest_font_name_length = len(f.Filename)
		}
	}
	return best_match_font_file_name
}

func font_name_contains(f *sysfont.Font, str string) bool {
	return strings.Contains(strings.ToLower(f.Name), strings.ToLower(str)) ||
		strings.Contains(strings.ToLower(f.Family), strings.ToLower(str)) ||
		strings.Contains(strings.ToLower(f.Filename), strings.ToLower(str))
}

func print_font_information(font *ttf.Font) {
	log_debug_msg("Family Name:", font.FaceFamilyName())
	log_debug_msg("Total Faces:", font.Faces())
	log_debug_msg("Ascent:", font.Ascent())
	log_debug_msg("Descent:", font.Descent())
	log_debug_msg("Height:", font.Height())
	log_debug_msg("FaceIsFixedWidth:", font.FaceIsFixedWidth())
	log_debug_msg("Outline:", font.GetOutline())
	log_debug_msg("LineSkip:", font.LineSkip())
	metrics, err := font.GlyphMetrics('M')
	if log_err_if_not_nil(err) {
		return
	}
	log_debug_msg("Metrics Advance:", metrics.Advance)
	log_debug_msg("Metrics MaxX:", metrics.MaxX)
	log_debug_msg("Metrics MinX:", metrics.MinX)
	log_debug_msg("Metrics MaxY:", metrics.MaxY)
	log_debug_msg("Metrics MinY:", metrics.MinY)
}
