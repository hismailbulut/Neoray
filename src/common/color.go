package common

import "fmt"

// We can't use generics for color types

type F32Color struct {
	R, G, B, A float32
}

func (c F32Color) String() string {
	return fmt.Sprintf("F32Color(R: %f, G: %f, B: %f, A: %f)", c.R, c.G, c.B, c.A)
}

type U8Color struct {
	R, G, B, A uint8
}

func (c U8Color) String() string {
	return fmt.Sprintf("F32Color(R: %d, G: %d, B: %d, A: %d)", c.R, c.G, c.B, c.A)
}

func (color U8Color) Pack() uint32 {
	rgb24 := uint32(color.R)
	rgb24 = (rgb24 << 8) | uint32(color.G)
	rgb24 = (rgb24 << 8) | uint32(color.B)
	return rgb24
}

func ColorFromUint(color uint32) U8Color {
	return U8Color{
		R: uint8((color >> 16) & 0xff),
		G: uint8((color >> 8) & 0xff),
		B: uint8(color & 0xff),
		A: 255,
	}
}

func (c U8Color) ToF32() F32Color {
	return F32Color{
		R: float32(c.R) / 255,
		G: float32(c.G) / 255,
		B: float32(c.B) / 255,
		A: float32(c.A) / 255,
	}
}
