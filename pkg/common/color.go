package common

import (
	"fmt"
)

// Zero values for reducing allocations
var (
	ZeroColorF32 = Color[float32]{}
	ZeroColorU8  = Color[uint8]{}
)

type Color[T Numbers] struct {
	R, G, B, A T
}

func (c Color[T]) String() string {
	return fmt.Sprintf("%T(R: %v, G: %v, B: %v, A: %v)", c, c.R, c.G, c.B, c.A)
}

// func (color Color[T]) Pack() uint32 {
//     rgb24 := uint32(color.R)
//     rgb24 = (rgb24 << 8) | uint32(color.G)
//     rgb24 = (rgb24 << 8) | uint32(color.B)
//     return rgb24
// }

func ColorFromUint(color uint32) Color[uint8] {
	return Color[uint8]{
		R: uint8((color >> 16) & 0xff),
		G: uint8((color >> 8) & 0xff),
		B: uint8(color & 0xff),
		A: 255,
	}
}

func (c Color[T]) ToF32() Color[float32] {
	return Color[float32]{
		R: float32(c.R) / 255,
		G: float32(c.G) / 255,
		B: float32(c.B) / 255,
		A: float32(c.A) / 255,
	}
}
