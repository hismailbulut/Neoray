package main

import (
	"fmt"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

type vec2 struct {
	X float32
	Y float32
}

type ivec2 struct {
	X int
	Y int
}

func convert_rgb24_to_rgba(color uint32) sdl.Color {
	return sdl.Color{
		// 0x000000rr & 0xff = red: 0xrr
		R: uint8((color >> 16) & 0xff),
		// 0x0000rrgg & 0xff = green: 0xgg
		G: uint8((color >> 8) & 0xff),
		// 0x00rrggbb & 0xff = blue: 0xbb
		B: uint8(color & 0xff),
		A: 255,
	}
}

func convert_rgba_to_rgb24(color sdl.Color) uint32 {
	// 0x00000000
	rgb24 := uint32(color.R)
	// 0x000000rr
	rgb24 = (rgb24 << 8) | uint32(color.G)
	// 0x0000rrgg
	rgb24 = (rgb24 << 8) | uint32(color.B)
	// 0x00rrggbb
	return rgb24
}

func is_color_black(color sdl.Color) bool {
	return color.R == 0 && color.G == 0 && color.B == 0
}

func measure_execution_time(name string) func() {
	now := time.Now()
	return func() {
		elapsed := time.Since(now)
		if elapsed.Milliseconds() > 1 {
			fmt.Println("Function", name, "takes", elapsed)
		}
	}
}
