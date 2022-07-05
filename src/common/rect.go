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

type Rectangle[T SignedNumbers] struct {
	X, Y, W, H T
}

func (rect Rectangle[T]) String() string {
	return fmt.Sprint("Rect(X: ", rect.X, ", Y: ", rect.Y, ", W: ", rect.W, ", H: ", rect.H, ")")
}

func Rect[T SignedNumbers](X, Y, W, H T) Rectangle[T] {
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
