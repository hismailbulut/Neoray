package main

import (
	"fmt"
	"time"

	rl "github.com/chunqian/go-raylib/raylib"
)

func convert_rgb24_to_rgba(color uint32) rl.Color {
	// 0x000000rr & 0xff = red: 0xrr
	// 0x0000rrgg & 0xff = green: 0xgg
	// 0x00rrggbb & 0xff = blue: 0xbb
	return rl.Color{
		R: uint8((color >> 16) & 0xff),
		G: uint8((color >> 8) & 0xff),
		B: uint8(color & 0xff),
		A: 255,
	}
}

func convert_rgba_to_rgb24(color rl.Color) uint32 {
	rgb24 := uint32(color.R)
	// 0x000000rr
	rgb24 = (rgb24 << 8) | uint32(color.G)
	// 0x0000rrgg
	rgb24 = (rgb24 << 8) | uint32(color.B)
	// 0x00rrggbb
	return rgb24
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
