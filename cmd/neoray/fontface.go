package main

import (
	"fmt"
	"image"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type FontFace struct {
	handle font.Face
	drawer font.Drawer

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
		handle:  face,
		loaded:  true,
		advance: advance.Ceil(),
		ascent:  face.Metrics().Ascent.Floor(),
		descent: face.Metrics().Descent.Floor(),
	}
	fontFace.drawer = font.Drawer{
		Src:  image.White,
		Face: fontFace.handle,
	}
	return fontFace, nil
}

func (fontFace *FontFace) RenderChar(c string) *image.RGBA {
	defer measure_execution_time("FontFace.RenderChar")()
	img := image.NewRGBA(image.Rect(0, 0, EditorSingleton.cellWidth, EditorSingleton.cellHeight))
	fontFace.drawer.Dst = img
	fontFace.drawer.Dot = fixed.P(0, EditorSingleton.cellHeight-fontFace.descent)
	// TODO: Check bounds
	fontFace.drawer.DrawString(c)
	return img
}
