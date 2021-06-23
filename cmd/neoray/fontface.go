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
	loaded bool

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
		// Maybe we should check for other glyphs
		// but I think every font has 'm'
		return FontFace{}, fmt.Errorf("Failed to get glyph advance!")
	}

	fontFace := FontFace{
		loaded:     true,
		handle:     face,
		fontHandle: sfont,
		advance:    advance.Ceil(),
		ascent:     face.Metrics().Ascent.Floor(),
		descent:    face.Metrics().Descent.Floor(),
		height:     face.Metrics().Height.Floor(),
	}

	return fontFace, nil
}

func (fontFace *FontFace) Resize(newsize float32) {
	if !fontFace.loaded {
		return
	}
	face, err := opentype.NewFace(fontFace.fontHandle, &opentype.FaceOptions{
		Size:    float64(newsize),
		DPI:     EditorSingleton.window.dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create new font face:", err)
		return
	}
	advance, ok := face.GlyphAdvance('m')
	if !ok {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to get glyph advance!")
		return
	}
	fontFace.handle = face
	fontFace.advance = advance.Ceil()
	fontFace.ascent = face.Metrics().Ascent.Floor()
	fontFace.descent = face.Metrics().Descent.Floor()
	fontFace.height = face.Metrics().Height.Floor()
}

func (fontFace *FontFace) IsDrawable(c string) bool {
	char := []rune(c)[0]
	i, err := fontFace.fontHandle.GlyphIndex(&fontFace.buffer, char)
	return i != 0 && err == nil
}

// This function draws horizontal line at given y coord.
// Don't use this function directly, use RenderChar instead.
func (fontFace *FontFace) drawLine(img *image.RGBA, y int) {
	for x := 0; x < img.Rect.Dx(); x++ {
		img.Set(x, y, color.White)
	}
}

// This function renders an undercurl to an empty image and returns it.
// The undercurl drawing job is done in the shaders.
// Feel free to change this function howewer you want to draw undercurl.
func (fontFace *FontFace) renderUndercurl() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, EditorSingleton.cellWidth, EditorSingleton.cellHeight))
	y := EditorSingleton.cellHeight - fontFace.descent
	for x := 0; x < img.Rect.Dx(); x++ {
		img.Set(x, y, color.White)
		// This evaluation will be true when x is not
		// center of a part (there are 4 parts) and x is not divisible by 3.
		// The 3 is for reducing count, we will do it for 2 of 3 times.
		// This makes curls more softer.
		if (x%4)%3 != 0 {
			// Divide image to 4 parts to get the number of the part.
			// If the part number is 0 or 2, then increase y, otherwise decrease it.
			if ((x/4)%4)%2 == 0 {
				y++
			} else {
				y--
			}
		}
	}
	return img
}

// Renders given rune and returns rendered RGBA image.
// Use RenderChar instead.
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

func (fontFace *FontFace) RenderChar(str string, underline, strikethrough bool) *image.RGBA {
	defer measure_execution_time("FontFace.RenderChar")()
	img := fontFace.renderGlyph([]rune(str)[0])
	if img == nil {
		return nil
	}
	if underline {
		fontFace.drawLine(img, EditorSingleton.cellHeight-fontFace.descent)
	}
	if strikethrough {
		fontFace.drawLine(img, EditorSingleton.cellHeight/2)
	}
	return img
}
