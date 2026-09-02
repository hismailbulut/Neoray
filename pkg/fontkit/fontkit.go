package fontkit

import (
	"fmt"

	"github.com/hismailbulut/Neoray/pkg/fontfinder"
)

// FontKit is a struct that holds different styles of same font family
// TODO: Maybe we should make this safe for concurrent usage, currently not
type FontKit struct {
	regular    *Font
	bold       *Font
	italic     *Font
	boldItalic *Font
}

func LoadFontKit(name string) (*FontKit, error) {
	info := fontfinder.Find(name)
	if info.Regular == "" && info.Bold == "" && info.Italic == "" && info.BoldItalic == "" {
		// This means we could not find any font file with this name
		return nil, fmt.Errorf("Couldn't find font %s", name)
	}
	kit := &FontKit{}
	// Load fonts
	var err error
	if info.Regular != "" {
		kit.regular, err = LoadFontFromFile(info.Regular)
		if err != nil {
			return nil, err
		}
	}
	if info.Bold != "" {
		kit.bold, err = LoadFontFromFile(info.Bold)
		if err != nil {
			return nil, err
		}
	}
	if info.Italic != "" {
		kit.italic, err = LoadFontFromFile(info.Italic)
		if err != nil {
			return nil, err
		}
	}
	if info.BoldItalic != "" {
		kit.boldItalic, err = LoadFontFromFile(info.BoldItalic)
		if err != nil {
			return nil, err
		}
	}
	return kit, nil
}

func LoadFontKitFromMemory(regular, bold, italic, boldItalic []byte) (*FontKit, error) {
	kit := &FontKit{}
	var err error
	kit.regular, err = LoadFontFromMem(regular)
	if err != nil {
		return nil, err
	}
	kit.bold, err = LoadFontFromMem(bold)
	if err != nil {
		return nil, err
	}
	kit.italic, err = LoadFontFromMem(italic)
	if err != nil {
		return nil, err
	}
	kit.boldItalic, err = LoadFontFromMem(boldItalic)
	if err != nil {
		return nil, err
	}
	return kit, nil
}

func (kit *FontKit) Regular() *Font {
	return kit.regular
}

func (kit *FontKit) Bold() *Font {
	return kit.bold
}

func (kit *FontKit) Italic() *Font {
	return kit.italic
}

func (kit *FontKit) BoldItalic() *Font {
	return kit.boldItalic
}

// Returns first non nil font starting from regular
func (kit *FontKit) FirstDrawableFont() *Font {
	if kit.Regular() != nil {
		return kit.Regular()
	}
	if kit.Bold() != nil {
		return kit.Bold()
	}
	if kit.Italic() != nil {
		return kit.Italic()
	}
	if kit.BoldItalic() != nil {
		return kit.BoldItalic()
	}
	return nil
}

func (kit *FontKit) SuitableFont(bold, italic bool) *Font {
	if bold && italic && kit.BoldItalic() != nil {
		return kit.BoldItalic()
	}
	if italic && kit.Italic() != nil {
		return kit.Italic()
	}
	if bold && kit.Bold() != nil {
		return kit.Bold()
	}
	if kit.Regular() != nil {
		return kit.Regular()
	}
	return nil
}

func (kit *FontKit) SuitableFontWithGlyph(bold, italic bool, char rune) *Font {
	if bold && italic && kit.BoldItalic() != nil {
		if kit.BoldItalic().ContainsGlyph(char) {
			return kit.BoldItalic()
		}
	}
	if italic && kit.Italic() != nil {
		if kit.Italic().ContainsGlyph(char) {
			return kit.Italic()
		}
	}
	if bold && kit.Bold() != nil {
		if kit.Bold().ContainsGlyph(char) {
			return kit.Bold()
		}
	}
	if kit.Regular() != nil {
		if kit.Regular().ContainsGlyph(char) {
			return kit.Regular()
		}
	}
	return nil
}

func (kit *FontKit) Unload() {
	if kit.regular != nil {
		kit.regular.Unload()
	}
	if kit.italic != nil {
		kit.italic.Unload()
	}
	if kit.bold != nil {
		kit.bold.Unload()
	}
	if kit.boldItalic != nil {
		kit.boldItalic.Unload()
	}
}
