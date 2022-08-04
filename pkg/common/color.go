package common

import (
	"fmt"
)

// Zero values for reducing allocations
var (
	ZeroColor = Color{}
)

type Color struct {
	R, G, B, A float32
}

func (c Color) String() string {
	return fmt.Sprintf("Color(R: %v, G: %v, B: %v, A: %v)", c.R, c.G, c.B, c.A)
}

func ColorFromUint(color uint32) Color {
	r := (color >> 16) & 0xff
	g := (color >> 8) & 0xff
	b := color & 0xff
	return Color{
		R: float32(r) / 255.0,
		G: float32(g) / 255.0,
		B: float32(b) / 255.0,
		A: 1.0,
	}
}
