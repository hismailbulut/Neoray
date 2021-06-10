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
	loaded bool

	advance int
	ascent  int
	descent int

	handle font.Face
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

	_, advance, ok := face.GlyphBounds('m')
	if !ok {
		return FontFace{}, fmt.Errorf("Failed to get glyph advance!")
	}

	return FontFace{
		loaded:  true,
		advance: advance.Ceil(),
		ascent:  face.Metrics().Ascent.Floor(),
		descent: face.Metrics().Descent.Floor(),
		handle:  face,
	}, nil
}

// TODO: Dont use image and drawer
func (fontFace *FontFace) RenderChar(c string) *image.RGBA {
	defer measure_execution_time("FontFace.RenderChar")()
	img := image.NewRGBA(image.Rect(0, 0, EditorSingleton.cellWidth, EditorSingleton.cellHeight))
	drawer := font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: fontFace.handle,
		Dot:  fixed.P(0, EditorSingleton.cellHeight-fontFace.descent),
	}
	drawer.DrawString(c)
	return img
}
