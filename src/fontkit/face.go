package fontkit

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/hismailbulut/neoray/src/common"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

type Face struct {
	handle          font.Face
	useBoxDrawing   bool
	useBlockDrawing bool
	// calculated
	size      float64
	thickness float32
	// metrics
	advance int
	ascent  int
	descent int
	height  int
}

func (f *Font) CreateFace(size, dpi float64, useBoxDrawing, useBlockDrawing bool) (*Face, error) {
	face := new(Face)
	face.useBoxDrawing = useBoxDrawing
	face.useBlockDrawing = useBlockDrawing
	var err error
	face.handle, err = opentype.NewFace(f.handle, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	{
		advance1, ok := face.handle.GlyphAdvance('m')
		if !ok {
			return nil, errors.New("Failed to get font advance!")
		}
		advance2, ok := face.handle.GlyphAdvance('i')
		if !ok {
			return nil, errors.New("Failed to get font advance!")
		}
		if advance1.Floor() != advance2.Floor() {
			// This font is not monospaced
			// experiment
			face.advance = (advance1.Floor() + advance2.Floor()) / 2
		} else {
			face.advance = advance1.Floor()
		}
	}
	metrics := face.handle.Metrics()
	face.ascent = metrics.Ascent.Ceil()
	face.descent = metrics.Descent.Floor()
	face.height = metrics.Height.Floor()

	face.size = size
	face.thickness = common.Max(float32(math.Ceil(4*(float64(face.height)/12))/4), 1)
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
	return rastDraw(r)
}

func (face *Face) DrawUnicodeBoxGlyph(char rune, imgSize common.Vector2[int]) *image.RGBA {
	light := face.thickness
	heavy := light * 2

	r := vector.NewRasterizer(imgSize.Width(), imgSize.Height())
	w := float32(imgSize.Width())
	h := float32(imgSize.Height())
	center := common.Vec2(w/2, h/2)

	switch char {
	case 0x2500: // light horizontal line
		rastLine(r, light, common.Vec2(0, h/2), common.Vec2(w, h/2))
	case 0x2501: // heavy horizontal line
		rastLine(r, heavy, common.Vec2(0, h/2), common.Vec2(w, h/2))
	case 0x2502: // light vertical line
		rastLine(r, light, common.Vec2(w/2, 0), common.Vec2(w/2, h))
	case 0x2503: // heavy vertical line
		rastLine(r, heavy, common.Vec2(w/2, 0), common.Vec2(w/2, h))

	case 0x250C, 0x250D, 0x250E, 0x250F,
		0x2510, 0x2511, 0x2512, 0x2513,
		0x2514, 0x2515, 0x2516, 0x2517,
		0x2518, 0x2519, 0x251A, 0x251B:
		n := char - 0x250C
		up := common.Vec2(w/2, h)
		if n/8 > 0 {
			up = common.Vec2(w/2, 0)
		}
		left := common.Vec2(w, h/2)
		if (n/4)%2 != 0 {
			left = common.Vec2(0, h/2)
		}
		upThickness := light
		if (n/2)%2 != 0 {
			upThickness = heavy
		}
		leftThickness := light
		if n%2 != 0 {
			leftThickness = heavy
		}
		rastCorner(r, center, up.ToVec3(upThickness), left.ToVec3(leftThickness))

	case 0x251C, 0x251D, 0x251E, 0x251F,
		0x2520, 0x2521, 0x2522, 0x2523,
		0x2524, 0x2525, 0x2526, 0x2527,
		0x2528, 0x2529, 0x252A, 0x252B:
		n := char - 0x251C
		right := common.Vec2(w, h/2)
		if n >= 8 {
			right = common.Vec2(0, h/2)
			n -= 8
		}
		upThickness := light
		if n == 2 || n == 4 || n == 5 || n == 7 {
			upThickness = heavy
		}
		rightThickness := light
		if n == 1 || n == 5 || n == 6 || n == 7 {
			rightThickness = heavy
		}
		downThickness := light
		if n == 3 || n == 4 || n == 6 || n == 7 {
			downThickness = heavy
		}
		rastCorner(r, center,
			common.Vec3(w/2, 0, upThickness),
			right.ToVec3(rightThickness),
			common.Vec3(w/2, h, downThickness))

	case 0x252C, 0x252D, 0x252E, 0x252F,
		0x2530, 0x2531, 0x2532, 0x2533,
		0x2534, 0x2535, 0x2536, 0x2537,
		0x2538, 0x2539, 0x253A, 0x253B:
		n := char - 0x252C
		down := common.Vec2(w/2, h)
		if n >= 8 {
			down = common.Vec2(w/2, 0)
		}
		leftThickness := light
		if n%2 != 0 {
			leftThickness = heavy
		}
		rightThickness := light
		if (n/2)%2 != 0 {
			rightThickness = heavy
		}
		downThickness := light
		if (n/4)%2 != 0 {
			downThickness = heavy
		}
		rastCorner(r, center,
			common.Vec3(0, h/2, leftThickness),
			common.Vec3(w, h/2, rightThickness),
			down.ToVec3(downThickness))

	case 0x253C, 0x253D, 0x253E, 0x253F,
		0x2540, 0x2541, 0x2542, 0x2543,
		0x2544, 0x2545, 0x2546, 0x2547,
		0x2548, 0x2549, 0x254A, 0x254B:
		n := char - 0x253C
		upThickness := light
		if n == 4 || n == 6 || n == 7 || n == 8 || n == 11 || n == 13 || n == 14 || n == 15 {
			upThickness = heavy
		}
		rightThickness := light
		if n == 2 || n == 3 || n == 8 || n == 10 || n == 11 || n == 12 || n == 14 || n == 15 {
			rightThickness = heavy
		}
		downThickness := light
		if n == 5 || n == 6 || n == 9 || n == 10 || n == 12 || n == 13 || n == 14 || n == 15 {
			downThickness = heavy
		}
		leftThickness := light
		if n == 1 || n == 3 || n == 7 || n == 9 || n == 11 || n == 12 || n == 13 || n == 15 {
			leftThickness = heavy
		}
		rastCorner(r, center,
			common.Vec3(w/2, 0, upThickness), common.Vec3(w, h/2, rightThickness),
			common.Vec3(w/2, h, downThickness), common.Vec3(0, h/2, leftThickness))

	// TODO: Doubles

	case 0x256D: // light down to right arc
		rastCurve(r, light, common.Vec2(w/2, h), common.Vec2(w, h/2), center)
	case 0x256E: // light down to left arc
		rastCurve(r, light, common.Vec2(w/2, h), common.Vec2(0, h/2), center)
	case 0x256F: // light up to left arc
		rastCurve(r, light, common.Vec2(w/2, 0), common.Vec2(0, h/2), center)
	case 0x2570: // light up to right arc
		rastCurve(r, light, common.Vec2(w/2, 0), common.Vec2(w, h/2), center)

	case 0x2571: // diagonal bot-left to top-right
		rastLine(r, light, common.Vec2(0, h), common.Vec2(w, 0))
	case 0x2572: // diagonal top-left to bot-right
		rastLine(r, light, common.Vec2[float32](0, 0), common.Vec2(w, h))
	case 0x2573: // both
		rastLine(r, light, common.Vec2(0, h), common.Vec2(w, 0))
		rastLine(r, light, common.Vec2[float32](0, 0), common.Vec2(w, h))

	case 0x2574, 0x2575, 0x2576, 0x2577,
		0x2578, 0x2579, 0x257A, 0x257B:
		n := char - 0x2574
		pos := common.Vec2(0, h/2)
		switch n % 4 {
		case 1: // up
			pos = common.Vec2(w/2, 0)
		case 2: // right
			pos = common.Vec2(w, h/2)
		case 3: // down
			pos = common.Vec2(w/2, h)
		}
		thickness := light
		if n/4 >= 1 {
			thickness = heavy
		}
		rastLine(r, thickness, pos, center)

	case 0x257C:
		rastLine(r, light, common.Vec2(0, h/2), center)
		rastLine(r, heavy, common.Vec2(w, h/2), center)
	case 0x257D:
		rastLine(r, light, common.Vec2(w/2, 0), center)
		rastLine(r, heavy, common.Vec2(w/2, h), center)
	case 0x257E:
		rastLine(r, heavy, common.Vec2(0, h/2), center)
		rastLine(r, light, common.Vec2(w, h/2), center)
	case 0x257F:
		rastLine(r, heavy, common.Vec2(w/2, 0), center)
		rastLine(r, light, common.Vec2(w/2, h), center)

	default:
		return nil
	}

	return rastDraw(r)
}

func (face *Face) DrawUnicodeBlockGlyph(char rune, imgSize common.Vector2[int]) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, imgSize.Width(), imgSize.Height()))
	w := float32(imgSize.Width())
	h := float32(imgSize.Height())

	switch char {
	case 0x2580: // upper 1/2 block
		drawRect(img, common.Rect(0, 0, w, h/2), 1)
	case 0x2581: // lower 1/8 block
		drawRect(img, common.Rect(0, h-h/8, w, h/8), 1)
	case 0x2582: // lower 1/4 block
		drawRect(img, common.Rect(0, h-h/4, w, h/4), 1)
	case 0x2583: // lower 3/8 block
		drawRect(img, common.Rect(0, h-(h*3/8), w, h*3/8), 1)
	case 0x2584: // lower 1/2 block
		drawRect(img, common.Rect(0, h/2, w, h/2), 1)
	case 0x2585: // lower 5/8 block
		drawRect(img, common.Rect(0, h-(h*5/8), w, h*5/8), 1)
	case 0x2586: // lower 3/4 block
		drawRect(img, common.Rect(0, h-(h*3/4), w, h*3/4), 1)
	case 0x2587: // lower 7/8 block
		drawRect(img, common.Rect(0, h-(h*7/8), w, h*7/8), 1)
	case 0x2588: // full block
		drawRect(img, common.Rect(0, 0, w, h), 1)
	case 0x2589: // left 7/8 block
		drawRect(img, common.Rect(0, 0, w*7/8, h), 1)
	case 0x258A: // left 3/4 block
		drawRect(img, common.Rect(0, 0, w*3/4, h), 1)
	case 0x258B: // left 5/8 block
		drawRect(img, common.Rect(0, 0, w*5/8, h), 1)
	case 0x258C: // left 1/2 block
		drawRect(img, common.Rect(0, 0, w/2, h), 1)
	case 0x258D: // left 3/8 block
		drawRect(img, common.Rect(0, 0, w*3/8, h), 1)
	case 0x258E: // left 1/4 block
		drawRect(img, common.Rect(0, 0, w/4, h), 1)
	case 0x258F: // left 1/8 block
		drawRect(img, common.Rect(0, 0, w/8, h), 1)
	case 0x2590: // rigt 1/2 block
		drawRect(img, common.Rect(w/2, 0, w/2, h), 1)
	case 0x2591: // light shade
		drawRect(img, common.Rect(0, 0, w, h), 0.25)
	case 0x2592: // medium shade
		drawRect(img, common.Rect(0, 0, w, h), 0.50)
	case 0x2593: // dark shade
		drawRect(img, common.Rect(0, 0, w, h), 0.75)
	case 0x2594: // upper 1/8 block
		drawRect(img, common.Rect(0, 0, w, h/8), 1)
	case 0x2595: // right 1/8 block
		drawRect(img, common.Rect(w-w/8, 0, w/8, h), 1)
	case 0x2596, 0x2597, 0x2598, 0x2599, 0x259A,
		0x259B, 0x259C, 0x259D, 0x259E, 0x259F: // quadrants
		n := char - 0x2596
		if n >= 2 && n <= 6 { // upper left
			drawRect(img, common.Rect(0, 0, w/2, h/2), 1)
		}
		if n == 0 || n == 3 || n == 5 || n == 8 || n == 9 { // lower left
			drawRect(img, common.Rect(0, h/2, w/2, h/2), 1)
		}
		if n >= 5 && n <= 9 { // upper right
			drawRect(img, common.Rect(w/2, 0, w/2, h/2), 1)
		}
		if n == 1 || n == 3 || n == 4 || n == 6 || n == 9 { // lower right
			drawRect(img, common.Rect(w/2, h/2, w/2, h/2), 1)
		}
	default:
		panic(fmt.Errorf("missing block glyph %d (%s)", char, string(char)))
	}

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
		if mask.Bounds().Dy() > height {
			// Center image if the image height is taller than our cell height.
			maskp = image.Pt(0, (height-mask.Bounds().Dy())/2)
		}
		img := image.NewRGBA(image.Rect(0, 0, width, height))
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
	w := float32(img.Rect.Dx())
	if underline {
		y := float32(imgSize.Height()-face.descent) + 1
		drawLine(img, common.Vec2(0, y), common.Vec2(w, y))
	}
	if strikethrough {
		y := float32(imgSize.Height()) / 2
		drawLine(img, common.Vec2(0, y), common.Vec2(w, y))
	}
	return img
}
