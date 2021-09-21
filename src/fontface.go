package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

type FontFace struct {
	handle     font.Face
	fontHandle *sfnt.Font
	buffer     sfnt.Buffer

	advance int
	ascent  int
	descent int
	height  int

	size      float64
	thickness float32
}

func CreateFace(fileName string, size float32) (*FontFace, error) {
	fileData, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file: %s\n", err)
	}
	return CreateFaceFromMem(fileData, size)
}

func CreateFaceFromMem(data []byte, size float32) (*FontFace, error) {
	sfont, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse font data: %s\n", err)
	}
	face := FontFace{
		fontHandle: sfont,
	}
	face.Resize(size)
	return &face, nil
}

func (face *FontFace) FamilyName() string {
	name, err := face.fontHandle.Name(&face.buffer, sfnt.NameIDFamily)
	if err != nil {
		logMessage(LEVEL_DEBUG, TYPE_NEORAY, "Failed to get family name of the font.")
	}
	return name
}

func (face *FontFace) Resize(newsize float32) {
	var err error
	face.handle, err = opentype.NewFace(face.fontHandle, &opentype.FaceOptions{
		Size:    float64(newsize),
		DPI:     singleton.window.dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		// This is actually impossible because opentype.NewFace always returns nil error.
		// But we will check anyway.
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to create a new font face:", err)
		return
	}
	face.size = float64(newsize)
	face.calcMetrics()
}

func (face *FontFace) calcMetrics() {
	advance, ok := face.handle.GlyphAdvance('m')
	if !ok {
		logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to get font advance!")
		return
	}
	face.advance = advance.Floor()
	metrics := face.handle.Metrics()
	face.ascent = metrics.Ascent.Ceil()
	face.descent = metrics.Descent.Floor()
	face.height = metrics.Height.Floor()

	face.thickness = f32max(float32(math.Ceil(4*(float64(face.height)/12))/4), 1)
}

// ContainsGlyph returns the whether font contains the given glyph.
func (face *FontFace) ContainsGlyph(char rune) bool {
	i, err := face.fontHandle.GlyphIndex(&face.buffer, char)
	return i != 0 && err == nil
}

// This function renders an undercurl to an empty image and returns it.
// The undercurl drawing job is done in the shaders.
func (face *FontFace) renderUndercurl() *image.RGBA {
	w := float32(singleton.cellWidth)
	h := float32(singleton.cellHeight)
	y := h - float32(face.descent)
	const xd = 10
	r := vector.NewRasterizer(singleton.cellWidth, singleton.cellHeight)
	thickness := f32max(face.thickness/2, 1)
	rastCurve(r, thickness, F32Vec2{0, y}, F32Vec2{w / 2, y}, F32Vec2{w / xd, h})
	rastCurve(r, thickness, F32Vec2{w / 2, y}, F32Vec2{w, y}, F32Vec2{w - (w / xd), h / 2})
	return rastDraw(r)
}

// Faster line, not antialiased and only 1 pixel
func drawLine(img *image.RGBA, begin, end F32Vec2) {
	// Round to pixels
	step := end.minus(begin).normalized()
	end_pix := end.toInt()
	point := begin
	for {
		pixel := point.toInt()
		img.Set(pixel.X, pixel.Y, color.White)
		if pixel.equals(end_pix) {
			break
		}
		point = point.plus(step)
	}
}

func drawRect(img *image.RGBA, rect F32Rect, alpha float32) {
	a := uint8(f32clamp(alpha, 0, 1) * 255)
	r := rect.toInt()
	for x := r.X; x <= r.X+r.W; x++ {
		for y := r.Y; y <= r.Y+r.H; y++ {
			img.SetRGBA(x, y, color.RGBA{255, 255, 255, a})
		}
	}
}

// Adds line operation to r
func rastLine(r *vector.Rasterizer, thickness float32, begin, end F32Vec2) {
	perp := end.minus(begin).perpendicular().normalized().multiplyS(thickness)
	perp_half := perp.divideS(2)

	begin = begin.minus(perp_half)
	end = end.minus(perp_half)

	r.MoveTo(begin.X, begin.Y)
	r.LineTo(end.X, end.Y)

	begin = begin.plus(perp)
	end = end.plus(perp)

	r.LineTo(end.X, end.Y)
	r.LineTo(begin.X, begin.Y)
	r.ClosePath()
}

// Z value of the vectors are thickness
func rastCorner(r *vector.Rasterizer, mid F32Vec2, points ...F32Vec3) {
	var boldest int = -1
	var boldest_thickness float32
	var boldest_horizontal bool
	for i, p := range points {
		if p.Z > boldest_thickness {
			boldest = i
			boldest_thickness = p.Z
			boldest_horizontal = p.toVec2().minus(mid).isHorizontal()
		}
	}
	var boldest2 int = -1
	var boldest2_thickness float32
	for i, p := range points {
		if i != boldest &&
			boldest_horizontal != p.toVec2().minus(mid).isHorizontal() &&
			p.Z > boldest2_thickness {
			boldest2 = i
			boldest2_thickness = p.Z
		}
	}
	assert(boldest >= 0 && boldest2 >= 0, "corner needs at least two points and there must be perpendicular vectors")
	for i, p := range points {
		m := mid
		vn := p.toVec2().minus(mid).normalized()
		if i == boldest {
			m = mid.minus(vn.multiplyS(boldest2_thickness / 2))
		} else if vn.isHorizontal() == boldest_horizontal {
			m = mid.plus(vn.multiplyS(boldest2_thickness / 2))
		} else {
			m = mid.plus(vn.multiplyS(boldest_thickness / 2))
		}
		rastLine(r, p.Z, m, p.toVec2())
	}
}

// Adds quadratic bezier curve operation to r
func rastCurve(r *vector.Rasterizer, thickness float32, begin, end, control F32Vec2) {
	bePerp := end.minus(begin).perpendicular().normalized().multiplyS(thickness)
	bcPerp := control.minus(begin).perpendicular().normalized().multiplyS(thickness)
	cePerp := end.minus(control).perpendicular().normalized().multiplyS(thickness)

	begin = begin.minus(bcPerp.divideS(2))
	end = end.minus(cePerp.divideS(2))
	control = control.minus(bePerp.divideS(2))

	r.MoveTo(begin.X, begin.Y)
	r.QuadTo(control.X, control.Y, end.X, end.Y)

	begin = begin.plus(bcPerp)
	end = end.plus(cePerp)
	control = control.plus(bePerp)

	r.LineTo(end.X, end.Y)
	r.QuadTo(control.X, control.Y, begin.X, begin.Y)
	r.ClosePath()
}

// draws rasterizer operations to an image and returns it
func rastDraw(r *vector.Rasterizer) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, r.Size().X, r.Size().Y))
	r.Draw(img, img.Rect, image.White, image.Point{})
	return img
}

func (face *FontFace) drawUnicodeBoxGlyph(char rune) *image.RGBA {
	light := face.thickness
	heavy := light * 2

	r := vector.NewRasterizer(singleton.cellWidth, singleton.cellHeight)
	w := float32(singleton.cellWidth)
	h := float32(singleton.cellHeight)
	center := F32Vec2{w / 2, h / 2}

	switch char {
	case 0x2500: // light horizontal line
		rastLine(r, light, F32Vec2{0, h / 2}, F32Vec2{w, h / 2})
	case 0x2501: // heavy horizontal line
		rastLine(r, heavy, F32Vec2{0, h / 2}, F32Vec2{w, h / 2})
	case 0x2502: // light vertical line
		rastLine(r, light, F32Vec2{w / 2, 0}, F32Vec2{w / 2, h})
	case 0x2503: // heavy vertical line
		rastLine(r, heavy, F32Vec2{w / 2, 0}, F32Vec2{w / 2, h})

	case 0x250C, 0x250D, 0x250E, 0x250F,
		0x2510, 0x2511, 0x2512, 0x2513,
		0x2514, 0x2515, 0x2516, 0x2517,
		0x2518, 0x2519, 0x251A, 0x251B:
		n := char - 0x250C
		up := F32Vec2{w / 2, h}
		if n/8 > 0 {
			up = F32Vec2{w / 2, 0}
		}
		left := F32Vec2{w, h / 2}
		if (n/4)%2 != 0 {
			left = F32Vec2{0, h / 2}
		}
		upThickness := light
		if (n/2)%2 != 0 {
			upThickness = heavy
		}
		leftThickness := light
		if n%2 != 0 {
			leftThickness = heavy
		}
		rastCorner(r, center, up.toVec3(upThickness), left.toVec3(leftThickness))

	case 0x251C, 0x251D, 0x251E, 0x251F,
		0x2520, 0x2521, 0x2522, 0x2523,
		0x2524, 0x2525, 0x2526, 0x2527,
		0x2528, 0x2529, 0x252A, 0x252B:
		n := char - 0x251C
		right := F32Vec2{w, h / 2}
		if n >= 8 {
			right = F32Vec2{0, h / 2}
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
			F32Vec3{w / 2, 0, upThickness},
			right.toVec3(rightThickness),
			F32Vec3{w / 2, h, downThickness})

	case 0x252C, 0x252D, 0x252E, 0x252F,
		0x2530, 0x2531, 0x2532, 0x2533,
		0x2534, 0x2535, 0x2536, 0x2537,
		0x2538, 0x2539, 0x253A, 0x253B:
		n := char - 0x252C
		down := F32Vec2{w / 2, h}
		if n >= 8 {
			down = F32Vec2{w / 2, 0}
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
			F32Vec3{0, h / 2, leftThickness},
			F32Vec3{w, h / 2, rightThickness},
			down.toVec3(downThickness))

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
			F32Vec3{w / 2, 0, upThickness}, F32Vec3{w, h / 2, rightThickness},
			F32Vec3{w / 2, h, downThickness}, F32Vec3{0, h / 2, leftThickness})

	// TODO: Doubles

	case 0x256D: // light down to right arc
		rastCurve(r, light, F32Vec2{w / 2, h}, F32Vec2{w, h / 2}, center)
	case 0x256E: // light down to left arc
		rastCurve(r, light, F32Vec2{w / 2, h}, F32Vec2{0, h / 2}, center)
	case 0x256F: // light up to left arc
		rastCurve(r, light, F32Vec2{w / 2, 0}, F32Vec2{0, h / 2}, center)
	case 0x2570: // light up to right arc
		rastCurve(r, light, F32Vec2{w / 2, 0}, F32Vec2{w, h / 2}, center)

	case 0x2571: // diagonal bot-left to top-right
		rastLine(r, light, F32Vec2{0, h}, F32Vec2{w, 0})
	case 0x2572: // diagonal top-left to bot-right
		rastLine(r, light, F32Vec2{0, 0}, F32Vec2{w, h})
	case 0x2573: // both
		rastLine(r, light, F32Vec2{0, h}, F32Vec2{w, 0})
		rastLine(r, light, F32Vec2{0, 0}, F32Vec2{w, h})

	case 0x2574, 0x2575, 0x2576, 0x2577,
		0x2578, 0x2579, 0x257A, 0x257B:
		n := char - 0x2574
		pos := F32Vec2{0, h / 2}
		switch n % 4 {
		case 1: // up
			pos = F32Vec2{w / 2, 0}
		case 2: // right
			pos = F32Vec2{w, h / 2}
		case 3: // down
			pos = F32Vec2{w / 2, h}
		}
		thickness := light
		if n/4 >= 1 {
			thickness = heavy
		}
		rastLine(r, thickness, pos, center)

	case 0x257C:
		rastLine(r, light, F32Vec2{0, h / 2}, center)
		rastLine(r, heavy, F32Vec2{w, h / 2}, center)
	case 0x257D:
		rastLine(r, light, F32Vec2{w / 2, 0}, center)
		rastLine(r, heavy, F32Vec2{w / 2, h}, center)
	case 0x257E:
		rastLine(r, heavy, F32Vec2{0, h / 2}, center)
		rastLine(r, light, F32Vec2{w, h / 2}, center)
	case 0x257F:
		rastLine(r, heavy, F32Vec2{w / 2, 0}, center)
		rastLine(r, light, F32Vec2{w / 2, h}, center)

	default:
		return nil
	}

	return rastDraw(r)
}

func (face *FontFace) drawUnicodeBlockGlyph(char rune) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, singleton.cellWidth, singleton.cellHeight))
	w := float32(singleton.cellWidth)
	h := float32(singleton.cellHeight)

	switch char {
	case 0x2580: // upper 1/2 block
		drawRect(img, F32Rect{0, 0, w, h / 2}, 1)
	case 0x2581: // lower 1/8 block
		drawRect(img, F32Rect{0, h - h/8, w, h / 8}, 1)
	case 0x2582: // lower 1/4 block
		drawRect(img, F32Rect{0, h - h/4, w, h / 4}, 1)
	case 0x2583: // lower 3/8 block
		drawRect(img, F32Rect{0, h - (h * 3 / 8), w, h * 3 / 8}, 1)
	case 0x2584: // lower 1/2 block
		drawRect(img, F32Rect{0, h / 2, w, h / 2}, 1)
	case 0x2585: // lower 5/8 block
		drawRect(img, F32Rect{0, h - (h * 5 / 8), w, h * 5 / 8}, 1)
	case 0x2586: // lower 3/4 block
		drawRect(img, F32Rect{0, h - (h * 3 / 4), w, h * 3 / 4}, 1)
	case 0x2587: // lower 7/8 block
		drawRect(img, F32Rect{0, h - (h * 7 / 8), w, h * 7 / 8}, 1)
	case 0x2588: // full block
		drawRect(img, F32Rect{0, 0, w, h}, 1)
	case 0x2589: // left 7/8 block
		drawRect(img, F32Rect{0, 0, w * 7 / 8, h}, 1)
	case 0x258A: // left 3/4 block
		drawRect(img, F32Rect{0, 0, w * 3 / 4, h}, 1)
	case 0x258B: // left 5/8 block
		drawRect(img, F32Rect{0, 0, w * 5 / 8, h}, 1)
	case 0x258C: // left 1/2 block
		drawRect(img, F32Rect{0, 0, w / 2, h}, 1)
	case 0x258D: // left 3/8 block
		drawRect(img, F32Rect{0, 0, w * 3 / 8, h}, 1)
	case 0x258E: // left 1/4 block
		drawRect(img, F32Rect{0, 0, w / 4, h}, 1)
	case 0x258F: // left 1/8 block
		drawRect(img, F32Rect{0, 0, w / 8, h}, 1)
	case 0x2590: // rigt 1/2 block
		drawRect(img, F32Rect{w / 2, 0, w / 2, h}, 1)
	case 0x2591: // light shade
		drawRect(img, F32Rect{0, 0, w, h}, 0.25)
	case 0x2592: // medium shade
		drawRect(img, F32Rect{0, 0, w, h}, 0.50)
	case 0x2593: // dark shade
		drawRect(img, F32Rect{0, 0, w, h}, 0.75)
	case 0x2594: // upper 1/8 block
		drawRect(img, F32Rect{0, 0, w, h / 8}, 1)
	case 0x2595: // right 1/8 block
		drawRect(img, F32Rect{w - w/8, 0, w / 8, h}, 1)
	case 0x2596, 0x2597, 0x2598, 0x2599, 0x259A,
		0x259B, 0x259C, 0x259D, 0x259E, 0x259F: // quadrants
		n := char - 0x2596
		if n >= 2 && n <= 6 { // upper left
			drawRect(img, F32Rect{0, 0, w / 2, h / 2}, 1)
		}
		if n == 0 || n == 3 || n == 5 || n == 8 || n == 9 { // lower left
			drawRect(img, F32Rect{0, h / 2, w / 2, h / 2}, 1)
		}
		if n >= 5 && n <= 9 { // upper right
			drawRect(img, F32Rect{w / 2, 0, w / 2, h / 2}, 1)
		}
		if n == 1 || n == 3 || n == 4 || n == 6 || n == 9 { // lower right
			drawRect(img, F32Rect{w / 2, h / 2, w / 2, h / 2}, 1)
		}
	default:
		logMessage(LEVEL_FATAL, TYPE_NEORAY, "missing block glyph:", char)
	}

	return img
}

// Renders given rune and returns rendered RGBA image.
// Width of the image is always equal to cellWidth or cellWidth*2
func (face *FontFace) renderGlyph(char rune) *image.RGBA {
	height := singleton.cellHeight
	dot := fixed.P(0, height-face.descent)
	dr, mask, maskp, _, ok := face.handle.Glyph(dot, char)
	if ok {
		width := singleton.cellWidth
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
func (face *FontFace) RenderChar(char rune, underline, strikethrough bool) *image.RGBA {
	if singleton.options.boxDrawingEnabled {
		if char >= 0x2500 && char <= 0x257F {
			// Unicode box drawing characters
			// https://www.compart.com/en/unicode/block/U+2500
			img := face.drawUnicodeBoxGlyph(char)
			if img != nil {
				return img
			}
		} else if char >= 0x2580 && char <= 0x259F {
			// Unicode block characters
			// https://www.compart.com/en/unicode/block/U+2580
			return face.drawUnicodeBlockGlyph(char)
		}
	}
	// Render glyph
	img := face.renderGlyph(char)
	if img == nil {
		return nil
	}
	// Draw underline or strikethrough to glyph
	w := float32(img.Rect.Dx())
	if underline {
		y := float32(singleton.cellHeight-face.descent) + 1
		drawLine(img, F32Vec2{0, y}, F32Vec2{w, y})
	}
	if strikethrough {
		y := float32(singleton.cellHeight) / 2
		drawLine(img, F32Vec2{0, y}, F32Vec2{w, y})
	}
	return img
}
