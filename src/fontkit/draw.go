package fontkit

import (
	"image"
	"image/color"

	"github.com/hismailbulut/neoray/src/common"
	"golang.org/x/image/vector"
)

// RECTANGLE OPERATIONS USED BY BOXDRAWING

func drawRect(img *image.RGBA, rect common.Rectangle[float32], alpha float32) {
	a := uint8(common.Clamp(alpha, 0, 1) * 255)
	r := rect.ToInt()
	for x := r.X; x <= r.X+r.W; x++ {
		for y := r.Y; y <= r.Y+r.H; y++ {
			img.SetRGBA(x, y, color.RGBA{255, 255, 255, a})
		}
	}
}

func drawRectLine(img *image.RGBA, begin, end common.Vector2[float32], thickness, alpha float32) {
	vec := end.Sub(begin)
	var rect common.Rectangle[float32]
	if vec.IsHorizontal() {
		onTheLeft := begin
		onTheRight := end
		if end.X < begin.X {
			onTheLeft = end
			onTheRight = begin
		}
		rect = common.Rectangle[float32]{
			X: onTheLeft.X,
			Y: onTheLeft.Y - thickness/2,
			W: onTheRight.X - onTheLeft.X,
			H: thickness,
		}
	} else {
		onTheTop := begin
		onTheBottom := end
		if end.Y < begin.Y {
			onTheTop = end
			onTheBottom = begin
		}
		rect = common.Rectangle[float32]{
			X: onTheTop.X - thickness/2,
			Y: onTheTop.Y,
			W: thickness,
			H: onTheBottom.Y - onTheTop.Y,
		}
	}
	drawRect(img, rect, alpha)
}

type pointWithThickness struct {
	vector    common.Vector2[float32]
	thickness float32
}

func drawRectLinesFromPoint(img *image.RGBA, alpha float32, mid common.Vector2[float32], points ...pointWithThickness) {
	dot := common.Vector2[float32]{}
	// Calculate the top left and bottom right corner of the dot rectangle while drawing the lines
	for _, p := range points {
		if p.vector.Sub(mid).IsHorizontal() {
			if p.thickness > dot.Y {
				dot.Y = p.thickness
			}
		} else {
			if p.thickness > dot.X {
				dot.X = p.thickness
			}
		}
		drawRectLine(img, mid, p.vector, p.thickness, alpha)
	}
	// Final dot will be the center rectangle and makes merged lines look better
	dotRect := common.Rectangle[float32]{
		X: mid.X - dot.X/2,
		Y: mid.Y - dot.Y/2,
		W: dot.X,
		H: dot.Y,
	}
	drawRect(img, dotRect, alpha)
}

type pointDouble struct {
	vector    common.Vector2[float32]
	thickness float32
	double    bool
}

func drawDoubleLinesFromPoint(img *image.RGBA, mid common.Vector2[float32], points ...pointDouble) {
	dot := common.Vector2[float32]{}
	// We will sort double and single lines, doubles first
	orderedPoints := make([]pointDouble, len(points))
	orderedPointsLeftIndex := 0
	orderedPointsRightIndex := len(points) - 1
	// Calculate the top left and bottom right corner of the dot rectangle while drawing the lines
	for _, p := range points {
		if p.vector.Sub(mid).IsHorizontal() {
			if p.double && p.thickness*3 > dot.Y {
				dot.Y = p.thickness * 3
			} else if p.thickness > dot.Y {
				dot.Y = p.thickness
			}
		} else {
			if p.double && p.thickness*3 > dot.X {
				dot.X = p.thickness * 3
			} else if p.thickness > dot.X {
				dot.X = p.thickness
			}
		}
		if p.double {
			orderedPoints[orderedPointsLeftIndex] = p
			orderedPointsLeftIndex++
		} else {
			orderedPoints[orderedPointsRightIndex] = p
			orderedPointsRightIndex--
		}
	}
	// Dot will be the center rectangle and makes merged lines look better
	dotRect := common.Rectangle[float32]{
		X: mid.X - dot.X/2,
		Y: mid.Y - dot.Y/2,
		W: dot.X,
		H: dot.Y,
	}
	if orderedPointsRightIndex == len(points)-1 { // This means there is no single line, every line is double
		// First draw double lines
		for _, p := range orderedPoints[:orderedPointsLeftIndex] {
			if !p.double {
				panic("order wrong")
			}
			// Draw 3 times heavier and clear center line
			drawRectLine(img, mid, p.vector, p.thickness*3, 1)
		}
		// Draw dot
		drawRect(img, dotRect, 1)
		// After the dot clear the centers of the double lines
		for _, p := range orderedPoints[:orderedPointsLeftIndex] {
			if !p.double {
				panic("order wrong")
			}
			drawRectLine(img, mid, p.vector, p.thickness, 0)
		}
		// Clear center of the dot
		dotClearRect := common.Rectangle[float32]{
			X: mid.X - dot.X/6,
			Y: mid.Y - dot.Y/6,
			W: dot.X / 3,
			H: dot.Y / 3,
		}
		drawRect(img, dotClearRect, 0)
	} else { // There are single and double lines
		for _, p := range orderedPoints {
			if p.double {
				// Draw 3 times heavier and clear center line
				drawRectLine(img, mid, p.vector, p.thickness*3, 1)
				// Clear center
				drawRectLine(img, mid, p.vector, p.thickness, 0)
			} else {
				drawRectLine(img, mid, p.vector, p.thickness, 1)
			}
		}
		// Finally draw dot rectangle
		drawRect(img, dotRect, 1)
	}
}

// VECTOR OPERATIONS

// Adds quadratic bezier curve operation to r
func rastCurve(r *vector.Rasterizer, thickness float32, begin, end, control common.Vector2[float32]) {
	bePerp := end.Sub(begin).Perpendicular().Normalized().MulS(thickness)
	bcPerp := control.Sub(begin).Perpendicular().Normalized().MulS(thickness)
	cePerp := end.Sub(control).Perpendicular().Normalized().MulS(thickness)

	begin = begin.Sub(bcPerp.DivS(2))
	end = end.Sub(cePerp.DivS(2))
	control = control.Sub(bePerp.DivS(2))

	r.MoveTo(begin.X, begin.Y)
	r.QuadTo(control.X, control.Y, end.X, end.Y)

	begin = begin.Add(bcPerp)
	end = end.Add(cePerp)
	control = control.Add(bePerp)

	r.LineTo(end.X, end.Y)
	r.QuadTo(control.X, control.Y, begin.X, begin.Y)
	r.ClosePath()
}

// Adds line operation to r
func rastLine(r *vector.Rasterizer, begin, end common.Vector2[float32], thickness float32) {
	perp := end.Sub(begin).Perpendicular().Normalized().MulS(thickness)
	perp_half := perp.DivS(2)

	begin = begin.Sub(perp_half)
	end = end.Sub(perp_half)

	r.MoveTo(begin.X, begin.Y)
	r.LineTo(end.X, end.Y)

	begin = begin.Add(perp)
	end = end.Add(perp)

	r.LineTo(end.X, end.Y)
	r.LineTo(begin.X, begin.Y)
	r.ClosePath()
}
