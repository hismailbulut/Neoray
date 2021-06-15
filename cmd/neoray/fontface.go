package main

import (
	"fmt"
	"image"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type FontFace struct {
	handle     font.Face
	fontHandle *sfnt.Font
	buffer     sfnt.Buffer

	loaded  bool
	advance int
	ascent  int
	descent int
}

func CreateFontFace(fileName string, size int) (FontFace, error) {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		return FontFace{}, fmt.Errorf("Failed to read file: %s\n", err)
	}

	sfont, err := opentype.Parse(fileData)
	if err != nil {
		return FontFace{}, fmt.Errorf("Failed to parse font data: %s\n", err)
	}

	face, err := opentype.NewFace(sfont, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return FontFace{}, fmt.Errorf("Failed to create font face: %s\n", err)
	}

	advance, ok := face.GlyphAdvance('m')
	if !ok {
		return FontFace{}, fmt.Errorf("Failed to get glyph advance!")
	}

	fontFace := FontFace{
		handle:     face,
		fontHandle: sfont,
		loaded:     true,
		advance:    advance.Ceil(),
		ascent:     face.Metrics().Ascent.Floor(),
		descent:    face.Metrics().Descent.Floor(),
	}

	return fontFace, nil
}

func (fontFace *FontFace) Resize(newsize int) error {
	face, err := opentype.NewFace(fontFace.fontHandle, &opentype.FaceOptions{
		Size:    float64(newsize),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return fmt.Errorf("Failed to create font face: %s\n", err)
	}
	advance, ok := face.GlyphAdvance('m')
	if !ok {
		return fmt.Errorf("Failed to get glyph advance!")
	}
	fontFace.handle = face
	fontFace.advance = advance.Ceil()
	fontFace.ascent = face.Metrics().Ascent.Floor()
	fontFace.descent = face.Metrics().Descent.Floor()
	return nil
}

func (fontFace *FontFace) IsDrawable(c string) bool {
	char := []rune(c)[0]
	i, err := fontFace.fontHandle.GlyphIndex(&fontFace.buffer, char)
	return i != 0 || err != nil
}

func (fontFace *FontFace) RenderChar(c string) *image.RGBA {
	defer measure_execution_time("FontFace.RenderChar")()
	img := image.NewRGBA(image.Rect(0, 0, EditorSingleton.cellWidth, EditorSingleton.cellHeight))
	drawer := font.Drawer{
		Src:  image.White,
		Dst:  img,
		Face: fontFace.handle,
		Dot:  fixed.P(0, EditorSingleton.cellHeight-fontFace.descent),
	}
	drawer.DrawString(c)
	return img
}
