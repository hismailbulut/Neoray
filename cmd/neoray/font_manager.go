package main

import (
	"github.com/hismailbulut/Neoray/pkg/fontkit"
	"github.com/hismailbulut/Neoray/pkg/logger"

	"github.com/hismailbulut/Neoray/cmd/neoray/assets"
)

type FontManager struct {
	editor    *Editor
	fontSize  float64
	fontNames []string
	kits      []*fontkit.FontKit
}

func NewFontManager(editor *Editor) *FontManager {
	fontManager := &FontManager{editor: editor}
	kit, err := fontkit.LoadFontKitFromMemory(assets.Regular, assets.Bold, assets.Italic, assets.BoldItalic)
	if err != nil {
		logger.Fatal("Failed to load default font:", err)
	}
	fontManager.Push(kit)
	return fontManager
}

// This is for setting global font size
func (fontManager *FontManager) SetFontSize(size float64) {
	fontManager.fontSize = size
	fontManager.editor.gridManager.SetGridFontSize(1, size)
	fontManager.editor.contextMenu.SetFontSize(size)
}

func (fontManager *FontManager) SetFontNames(names []string) {
	fontManager.fontNames = names
	// TODO: do not remove all fonts if they are same
	// just remove the ones not presented, add new ones and rearrange order
	if fontManager.editor.state >= EditorWindowShown {
		fontManager.LoadFonts()
	}
}

func (fontManager *FontManager) LoadFonts() {
	// clear the font list to release previous fonts
	fontManager.Clear()
	// Create and set font
	for _, fontName := range fontManager.fontNames {
		logger.Trace("Loading font", fontName)
		kit, err := fontkit.LoadFontKit(fontName)
		if err != nil {
			fontManager.editor.nvim.EchoError("Font %s not found", fontName)
		} else {
			// Log some info
			if kit.Regular() != nil {
				logger.Trace("Regular:", kit.Regular().FilePath())
			}
			if kit.Bold() != nil {
				logger.Trace("Bold:", kit.Bold().FilePath())
			}
			if kit.Italic() != nil {
				logger.Trace("Italic:", kit.Italic().FilePath())
			}
			if kit.BoldItalic() != nil {
				logger.Trace("BoldItalic:", kit.BoldItalic().FilePath())
			}
			// Push this fonts
			fontManager.Push(kit)
		}
	}
	fontManager.editor.MarkForceDraw()
}

func (fontManager *FontManager) Clear() {
	fontManager.UnloadFonts()
	fontManager.editor.MarkForceDraw()
}

func (fontManager *FontManager) Push(kit *fontkit.FontKit) {
	fontManager.kits = append(fontManager.kits, kit)
	fontManager.editor.MarkForceDraw()
}

func (fontManager *FontManager) DefaultFont(bold, italic bool) *fontkit.Font {
	return fontManager.kits[0].SuitableFont(bold, italic)
}

// if char is less than 0 function will not look for any character
func (fontManager *FontManager) SuitableFont(bold, italic bool, char rune) *fontkit.Font {
	for i := len(fontManager.kits) - 1; i >= 0; i-- {
		kit := fontManager.kits[i]
		var font *fontkit.Font
		if char >= 0 { // character is important
			font = kit.SuitableFontWithGlyph(bold, italic, char)
		} else { // character is not important
			font = kit.SuitableFont(bold, italic)
		}
		if font != nil {
			if i < len(fontManager.kits)-1 {
				// logger.Debug("Font fallback to index", i, "for glyph", string(char))
			}
			return font
		}
	}
	return nil
}

func (fontManager *FontManager) MustSuitableFont(bold, italic bool, char rune) *fontkit.Font {
	font := fontManager.SuitableFont(bold, italic, char)
	if font != nil {
		return font
	}
	return fontManager.DefaultFont(bold, italic)
}

func (fontManager *FontManager) UnloadFonts() {
	for i := 1; i < len(fontManager.kits); i++ {
		fontManager.kits[i].Unload()
	}
	fontManager.kits = fontManager.kits[:1]
}

func (fontManager *FontManager) Destroy() {
	fontManager.UnloadFonts()
	fontManager.kits[0].Unload() // Default font kit (this is unnecessary for now)
	fontManager.kits = nil
}
