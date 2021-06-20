package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

type FontFace struct {
	size float32

	handle     font.Face
	fontHandle *sfnt.Font
	buffer     sfnt.Buffer

	advance int
	ascent  int
	descent int
	height  int
}

func CreateFontFace(fileName string, size float32) (FontFace, error) {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		return FontFace{}, fmt.Errorf("Failed to read file: %s\n", err)
	}
	return CreateFontFaceMemory(fileData, size)
}

func CreateFontFaceMemory(data []byte, size float32) (FontFace, error) {
	sfont, err := opentype.Parse(data)
	if err != nil {
		return FontFace{}, fmt.Errorf("Failed to parse font data: %s\n", err)
	}

	face, err := opentype.NewFace(sfont, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     EditorSingleton.window.dpi,
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
		size:       size,
		handle:     face,
		fontHandle: sfont,
		advance:    advance.Ceil(),
		ascent:     face.Metrics().Ascent.Floor(),
		descent:    face.Metrics().Descent.Floor(),
		height:     face.Metrics().Height.Floor(),
	}

	return fontFace, nil
}

func (fontFace *FontFace) Resize(newsize float32) error {
	if newsize == fontFace.size {
		return fmt.Errorf("Already same size!")
	}
	face, err := opentype.NewFace(fontFace.fontHandle, &opentype.FaceOptions{
		Size:    float64(newsize),
		DPI:     EditorSingleton.window.dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return fmt.Errorf("Failed to create font face: %s\n", err)
	}
	advance, ok := face.GlyphAdvance('m')
	if !ok {
		return fmt.Errorf("Failed to get glyph advance!")
	}
	fontFace.size = newsize
	fontFace.handle = face
	fontFace.advance = advance.Ceil()
	fontFace.ascent = face.Metrics().Ascent.Floor()
	fontFace.descent = face.Metrics().Descent.Floor()
	fontFace.height = face.Metrics().Height.Floor()
	return nil
}

func (fontFace *FontFace) IsDrawable(c string) bool {
	char := []rune(c)[0]
	i, err := fontFace.fontHandle.GlyphIndex(&fontFace.buffer, char)
	return i != 0 || err != nil
}

// Use RenderChar
func (fontFace *FontFace) renderUnderline(img *image.RGBA) {
	y := EditorSingleton.cellHeight - fontFace.descent
	for x := 0; x < img.Rect.Dx(); x++ {
		img.Set(x, y, color.White)
	}
}

// TODO
func (fontFace *FontFace) renderUndercurl(img *image.RGBA) {
	y := EditorSingleton.cellHeight - fontFace.descent
	diff := img.Rect.Dy() - y
	for x := 0; x < img.Rect.Dx(); x++ {
		img.Set(x, y, color.RGBA{R: 255, G: 1, B: 1, A: 255})
		if x%diff < diff/2 {
			y++
		} else if x%diff > diff/2 {
			y--
		}
	}
}

// Use RenderChar
func (fontFace *FontFace) renderGlyph(c rune) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, EditorSingleton.cellWidth, EditorSingleton.cellHeight))
	dot := fixed.P(0, EditorSingleton.cellHeight-fontFace.descent)
	dr, mask, maskp, _, ok := fontFace.handle.Glyph(dot, c)
	if ok {
		draw.DrawMask(img, dr, image.White, image.Point{}, mask, maskp, draw.Over)
		return img
	}
	return nil
}

func (fontFace *FontFace) RenderChar(str string, underline bool) *image.RGBA {
	defer measure_execution_time("FontFace.RenderChar")()
	img := fontFace.renderGlyph([]rune(str)[0])
	if img == nil {
		return nil
	}
	if underline {
		fontFace.renderUnderline(img)
	}
	return img
}
