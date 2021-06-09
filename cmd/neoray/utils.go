package main

import (
	"strings"
	"sync/atomic"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	COLOR_WHITE       = sdl.Color{R: 255, G: 255, B: 255, A: 255}
	COLOR_BLACK       = sdl.Color{R: 0, G: 0, B: 0, A: 255}
	COLOR_TRANSPARENT = sdl.Color{R: 0, G: 0, B: 0, A: 0}
)

type f32vec2 struct {
	X, Y float32
}

type i32vec2 struct {
	X, Y int32
}

type ivec2 struct {
	X, Y int
}

type f32color struct {
	R float32
	G float32
	B float32
	A float32
}

func packColor(color sdl.Color) uint32 {
	rgb24 := uint32(color.R)
	rgb24 = (rgb24 << 8) | uint32(color.G)
	rgb24 = (rgb24 << 8) | uint32(color.B)
	return rgb24
}

func unpackColor(color uint32) sdl.Color {
	return sdl.Color{
		R: uint8((color >> 16) & 0xff),
		G: uint8((color >> 8) & 0xff),
		B: uint8(color & 0xff),
		A: 255,
	}
}

func color_u8_to_f32(sdlcolor sdl.Color) f32color {
	return f32color{
		R: float32(sdlcolor.R) / 256,
		G: float32(sdlcolor.G) / 256,
		B: float32(sdlcolor.B) / 256,
		A: float32(sdlcolor.A) / 256,
	}
}

func colorIsBlack(color sdl.Color) bool {
	return color.R == 0 && color.G == 0 && color.B == 0
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func triangulateRect(rect sdl.Rect) [4]f32vec2 {
	return [4]f32vec2{
		{float32(rect.X), float32(rect.Y)},                   //0
		{float32(rect.X), float32(rect.Y + rect.H)},          //1
		{float32(rect.X + rect.W), float32(rect.Y + rect.H)}, //2
		{float32(rect.X + rect.W), float32(rect.Y)},          //3
	}
}

func triangulateFRect(rect sdl.FRect) [4]f32vec2 {
	return [4]f32vec2{
		{rect.X, rect.Y},                   //0
		{rect.X, rect.Y + rect.H},          //1
		{rect.X + rect.W, rect.Y + rect.H}, //2
		{rect.X + rect.W, rect.Y},          //3
	}
}

func ortho(top, left, right, bottom, near, far float32) [16]float32 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	return [16]float32{
		float32(2. / rml), 0, 0, 0, // 1
		0, float32(2. / tmb), 0, 0, // 2
		0, 0, float32(-2. / fmn), 0, // 3
		float32(-(right + left) / rml), // 4
		float32(-(top + bottom) / tmb),
		float32(-(far + near) / fmn), 1}
}

func afterSubstr(str string, substrs ...string) string {
	for _, substr := range substrs {
		idx := strings.Index(str, substr)
		if idx != -1 {
			return str[idx+len(substr):]
		}
	}
	return ""
}

func beginsWith(str string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Index(str, substr) == 0 {
			return true
		}
	}
	return false
}

func has_flag_u16(val, flag uint16) bool {
	return val&flag != 0
}

func atomicSetBool(b *int32, value bool) {
	var val int32 = 0
	if value == true {
		val = 1
	}
	atomic.StoreInt32(b, val)
}

func atomicGetBool(b *int32) bool {
	val := atomic.LoadInt32(b)
	if val == 0 {
		return false
	}
	return true
}
