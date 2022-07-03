package fontkit

import (
	"image"
	"image/color"

	"github.com/hismailbulut/neoray/src/common"
	"golang.org/x/image/vector"
)

// Faster line, not antialiased and only 1 pixel
func drawLine(img *image.RGBA, begin, end common.Vector2[float32]) {
	// Round to pixels
	step := end.Minus(begin).Normalized()
	end_pix := end.ToInt()
	point := begin
	for {
		pixel := point.ToInt()
		img.Set(pixel.X, pixel.Y, color.White)
		if pixel.Equals(end_pix) {
			break
		}
		point = point.Plus(step)
	}
}

func drawRect(img *image.RGBA, rect common.Rectangle[float32], alpha float32) {
	a := uint8(common.Clamp(alpha, 0, 1) * 255)
	r := rect.ToInt()
	for x := r.X; x <= r.X+r.W; x++ {
		for y := r.Y; y <= r.Y+r.H; y++ {
			img.SetRGBA(x, y, color.RGBA{255, 255, 255, a})
		}
	}
}

// Adds line operation to r
func rastLine(r *vector.Rasterizer, thickness float32, begin, end common.Vector2[float32]) {
	perp := end.Minus(begin).Perp().Normalized().MultiplyScalar(thickness)
	perp_half := perp.DivideScalar(2)

	begin = begin.Minus(perp_half)
	end = end.Minus(perp_half)

	r.MoveTo(begin.X, begin.Y)
	r.LineTo(end.X, end.Y)

	begin = begin.Plus(perp)
	end = end.Plus(perp)

	r.LineTo(end.X, end.Y)
	r.LineTo(begin.X, begin.Y)
	r.ClosePath()
}

// Z value of the vectors are thickness
func rastCorner(r *vector.Rasterizer, mid common.Vector2[float32], points ...common.Vector3[float32]) {
	var boldest int = -1
	var boldest_thickness float32
	var boldest_horizontal bool
	for i, p := range points {
		if p.Z > boldest_thickness {
			boldest = i
			boldest_thickness = p.Z
			boldest_horizontal = p.ToVec2().Minus(mid).IsHorizontal()
		}
	}
	var boldest2 int = -1
	var boldest2_thickness float32
	for i, p := range points {
		if i != boldest &&
			boldest_horizontal != p.ToVec2().Minus(mid).IsHorizontal() &&
			p.Z > boldest2_thickness {
			boldest2 = i
			boldest2_thickness = p.Z
		}
	}
	if boldest < 0 || boldest2 < 0 {
		panic("corner needs at least two points and there must be perpendicular vectors")
	}
	for i, p := range points {
		m := mid
		vn := p.ToVec2().Minus(mid).Normalized()
		if i == boldest {
			m = mid.Minus(vn.MultiplyScalar(boldest2_thickness / 2))
		} else if vn.IsHorizontal() == boldest_horizontal {
			m = mid.Plus(vn.MultiplyScalar(boldest2_thickness / 2))
		} else {
			m = mid.Plus(vn.MultiplyScalar(boldest_thickness / 2))
		}
		rastLine(r, p.Z, m, p.ToVec2())
	}
}

// Adds quadratic bezier curve operation to r
func rastCurve(r *vector.Rasterizer, thickness float32, begin, end, control common.Vector2[float32]) {
	bePerp := end.Minus(begin).Perp().Normalized().MultiplyScalar(thickness)
	bcPerp := control.Minus(begin).Perp().Normalized().MultiplyScalar(thickness)
	cePerp := end.Minus(control).Perp().Normalized().MultiplyScalar(thickness)

	begin = begin.Minus(bcPerp.DivideScalar(2))
	end = end.Minus(cePerp.DivideScalar(2))
	control = control.Minus(bePerp.DivideScalar(2))

	r.MoveTo(begin.X, begin.Y)
	r.QuadTo(control.X, control.Y, end.X, end.Y)

	begin = begin.Plus(bcPerp)
	end = end.Plus(cePerp)
	control = control.Plus(bePerp)

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
