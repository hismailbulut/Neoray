package common

import (
	"fmt"
	"math"
)

// Zero values for reducing allocations
var (
	ZeroRectangleF32 = Rectangle[float32]{}
	ZeroRectangleINT = Rectangle[int]{}
)

type Rectangle[T Numbers] struct {
	X, Y, W, H T
}

func (rect Rectangle[T]) String() string {
	return fmt.Sprintf("%T(X: %v, Y: %v, W: %v, H: %v)", rect, rect.X, rect.Y, rect.W, rect.H)
}

// Shortcut for creating a new rectangle
func Rect[T Numbers](X, Y, W, H T) Rectangle[T] {
	return Rectangle[T]{X: X, Y: Y, W: W, H: H}
}

func (rect Rectangle[T]) ToInt() Rectangle[int] {
	return Rectangle[int]{
		X: int(math.Floor(float64(rect.X))),
		Y: int(math.Floor(float64(rect.Y))),
		W: int(math.Floor(float64(rect.W))),
		H: int(math.Floor(float64(rect.H))),
	}
}
