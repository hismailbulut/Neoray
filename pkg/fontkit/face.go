package fontkit

import (
	"errors"
	"image"
	"image/draw"
	"math"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

type FaceParams struct {
	Size, DPI                      float64
	UseBoxDrawing, UseBlockDrawing bool
}

type Face struct {
	handle font.Face
	// config
	useBoxDrawing   bool
	useBlockDrawing bool
	// metrics
	advance int
	ascent  int
	descent int
	height  int
	// calculated
	thickness float32
	// cache
	imgCache map[common.Vector2[int]]*image.RGBA
}

// Do not use this function directly
// Use Font.CreateFace instead
func NewFace(f *sfnt.Font, params FaceParams) (*Face, error) {
	face := &Face{
		useBoxDrawing:   params.UseBoxDrawing,
		useBlockDrawing: params.UseBlockDrawing,
	}

	var err error
	face.handle, err = opentype.NewFace(f, &opentype.FaceOptions{
		Size:    params.Size,
		DPI:     params.DPI,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	// All fonts must have a space character and space character's width is set to same advance
	// value with all glyphs for a monospaced font
	advance, ok := face.handle.GlyphAdvance(' ')
	if !ok {
		return nil, errors.New("failed to get font advance")
	}
	face.advance = advance.Floor()

	metrics := face.handle.Metrics()
	face.ascent = metrics.Ascent.Ceil()
	face.descent = metrics.Descent.Floor()
	face.height = metrics.Height.Floor()

	face.thickness = common.Max(float32(math.Ceil(4*(float64(face.height)/12))/4), 1)
	face.imgCache = make(map[common.Vector2[int]]*image.RGBA)

	return face, nil
}

func (face *Face) ImageSize() common.Vector2[int] {
	return common.Vec2(face.advance, face.height)
}

// This function renders an undercurl to an empty image and returns it.
// The undercurl drawing job is done in the shaders.
func (face *Face) RenderUndercurl(imgSize common.Vector2[int]) *image.RGBA {
	w := float32(imgSize.Width())
	h := float32(imgSize.Height())
	y := h - float32(face.descent)/2
	r := vector.NewRasterizer(imgSize.Width(), imgSize.Height())
	rastCurve(r, face.thickness,
		common.Vec2(0, y),
		common.Vec2(w/2, y),
		common.Vec2(w/4, y+h/8),
	)
	rastCurve(r, face.thickness,
		common.Vec2(w/2, y),
		common.Vec2(w, y),
		common.Vec2(w/4*3, y-h/8),
	)
	img := face.cachedImage(imgSize)
	r.Draw(img, img.Rect, image.White, image.Point{})
	return img
}

// This function reduces image allocations by reusing existing sized images
func (face *Face) cachedImage(imgSize common.Vector2[int]) *image.RGBA {
	img, ok := face.imgCache[imgSize]
	if ok {
		// Clear image pixels
		for i := range img.Pix {
			img.Pix[i] = 0
		}
		return img
	}
	// Create new image
	img = image.NewRGBA(image.Rect(0, 0, imgSize.Width(), imgSize.Height()))
	// Cache image
	face.imgCache[imgSize] = img
	return img
}

// Renders given rune and returns rendered RGBA image.
// Width of the image is always equal to cellWidth or cellWidth*2
func (face *Face) RenderGlyph(char rune, imgSize common.Vector2[int]) *image.RGBA {
	height := imgSize.Height()
	dot := fixed.P(0, height-face.descent)
	dr, mask, maskp, _, ok := face.handle.Glyph(dot, char)
	if ok {
		width := imgSize.Width()
		if mask.Bounds().Dx() > width {
			width *= 2
		}
		if mask.Bounds().Dx() > width {
			logger.Error("Too wide character:", string(char), char)
		}
		if mask.Bounds().Dy() > height {
			// Center image if the image height is taller than our cell height.
			maskp = image.Pt(0, (height-mask.Bounds().Dy())/2)
		}
		img := face.cachedImage(common.Vec2(width, height))
		draw.DrawMask(img, dr, image.White, image.Point{}, mask, maskp, draw.Over)
		return img
	}
	return nil
}

// Renders given char to an RGBA image and returns.
// Also renders underline and strikethrough if specified.
func (face *Face) RenderChar(char rune, underline, strikethrough bool, imgSize common.Vector2[int]) *image.RGBA {
	if face.useBoxDrawing && char >= 0x2500 && char <= 0x257F {
		// Unicode box drawing characters
		// https://www.compart.com/en/unicode/block/U+2500
		img := face.DrawUnicodeBoxGlyph(char, imgSize)
		if img != nil {
			return img
		}
	}
	if face.useBlockDrawing && char >= 0x2580 && char <= 0x259F {
		// Unicode block characters
		// https://www.compart.com/en/unicode/block/U+2580
		return face.DrawUnicodeBlockGlyph(char, imgSize)
	}
	// Render glyph
	img := face.RenderGlyph(char, imgSize)
	if img == nil {
		return nil
	}
	// Draw underline or strikethrough to glyph
	if underline || strikethrough {
		w := float32(img.Rect.Dx())
		r := vector.NewRasterizer(img.Rect.Dx(), img.Rect.Dy())
		if underline {
			y := float32(imgSize.Height()-face.descent) + 1
			rastLine(r, common.Vec2(0, y), common.Vec2(w, y), face.thickness)
		}
		if strikethrough {
			y := float32(imgSize.Height()) / 2
			rastLine(r, common.Vec2(0, y), common.Vec2(w, y), face.thickness)
		}
		r.Draw(img, img.Rect, image.White, image.Point{})
	}
	return img
}
