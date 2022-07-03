package fontkit

import (
	"fmt"

	"github.com/hismailbulut/neoray/src/caskaydia"
	"github.com/hismailbulut/neoray/src/fontfinder"
)

var defaultFontKit *FontKit

// FontKit is a struct that holds different styles of same font family
// TODO: Maybe we should make this safe for concurrent usage, currently not
type FontKit struct {
	regular    *Font
	bold       *Font
	italic     *Font
	boldItalic *Font
}

func CreateKit(fontname string) (*FontKit, error) {
	info := fontfinder.Find(fontname)
	if info.Regular == "" && info.Bold == "" && info.Italic == "" && info.BoldItalic == "" {
		// This means we could not find any font file with this name
		return nil, fmt.Errorf("Couldn't find font %s", fontname)
	}
	fontkit := new(FontKit)
	// Load fonts
	var err error
	if info.Regular != "" {
		fontkit.regular, err = CreateFontFromFile(info.Regular)
		if err != nil {
			return nil, err
		}
	}
	if info.Bold != "" {
		fontkit.bold, err = CreateFontFromFile(info.Bold)
		if err != nil {
			return nil, err
		}
	}
	if info.Italic != "" {
		fontkit.italic, err = CreateFontFromFile(info.Italic)
		if err != nil {
			return nil, err
		}
	}
	if info.BoldItalic != "" {
		fontkit.boldItalic, err = CreateFontFromFile(info.BoldItalic)
		if err != nil {
			return nil, err
		}
	}
	return fontkit, nil
}

// Returns default font kit, creates it if first time
func Default() *FontKit {
	if defaultFontKit == nil {
		defaultFontKit = new(FontKit)
		defaultFontKit.regular, _ = CreateFontFromMem(caskaydia.Regular)
		defaultFontKit.bold, _ = CreateFontFromMem(caskaydia.Bold)
		defaultFontKit.italic, _ = CreateFontFromMem(caskaydia.Italic)
		defaultFontKit.boldItalic, _ = CreateFontFromMem(caskaydia.BoldItalic)
	}
	return defaultFontKit
}

func (fontkit *FontKit) Regular() *Font {
	return fontkit.regular
}

func (fontkit *FontKit) Bold() *Font {
	return fontkit.bold
}

func (fontkit *FontKit) Italic() *Font {
	return fontkit.italic
}

func (fontkit *FontKit) BoldItalic() *Font {
	return fontkit.boldItalic
}

// Returns first non nil font starting from regular
func (fontkit *FontKit) DefaultFont() *Font {
	if fontkit.regular != nil {
		return fontkit.regular
	}
	if fontkit.bold != nil {
		return fontkit.bold
	}
	if fontkit.italic != nil {
		return fontkit.italic
	}
	if fontkit.boldItalic != nil {
		return fontkit.boldItalic
	}
	panic("all fonts are nil")
}

func (fontkit *FontKit) SuitableFont(bold, italic bool) *Font {
	if bold && italic && fontkit.boldItalic != nil {
		return fontkit.boldItalic
	}
	if italic && fontkit.italic != nil {
		return fontkit.italic
	}
	if bold && fontkit.bold != nil {
		return fontkit.bold
	}
	if fontkit.regular != nil {
		return fontkit.regular
	}
	return fontkit.DefaultFont()
}
