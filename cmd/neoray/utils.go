package main

import (
	"math"
	"sync/atomic"
)

var (
	COLOR_WHITE       = U8Color{R: 255, G: 255, B: 255, A: 255}
	COLOR_BLACK       = U8Color{R: 0, G: 0, B: 0, A: 255}
	COLOR_TRANSPARENT = U8Color{R: 0, G: 0, B: 0, A: 0}
)

type U8Color struct {
	R, G, B, A uint8
}

type F32Color struct {
	R, G, B, A float32
}

type IntVec2 struct {
	X, Y int
}

type F32Vec2 struct {
	X, Y float32
}

type IntRect struct {
	X, Y, W, H int
}

type F32Rect struct {
	X, Y, W, H float32
}

func packColor(color U8Color) uint32 {
	rgb24 := uint32(color.R)
	rgb24 = (rgb24 << 8) | uint32(color.G)
	rgb24 = (rgb24 << 8) | uint32(color.B)
	return rgb24
}

func unpackColor(color uint32) U8Color {
	return U8Color{
		R: uint8((color >> 16) & 0xff),
		G: uint8((color >> 8) & 0xff),
		B: uint8(color & 0xff),
		A: 255,
	}
}

func (c U8Color) ToF32Color() F32Color {
	return F32Color{
		R: float32(c.R) / 256,
		G: float32(c.G) / 256,
		B: float32(c.B) / 256,
		A: float32(c.A) / 256,
	}
}

func colorIsBlack(color U8Color) bool {
	return color.R == 0 && color.G == 0 && color.B == 0
}

func triangulateRect(rect IntRect) [4]F32Vec2 {
	return [4]F32Vec2{
		{float32(rect.X), float32(rect.Y)},                   //0
		{float32(rect.X), float32(rect.Y + rect.H)},          //1
		{float32(rect.X + rect.W), float32(rect.Y + rect.H)}, //2
		{float32(rect.X + rect.W), float32(rect.Y)},          //3
	}
}

func triangulateFRect(rect F32Rect) [4]F32Vec2 {
	return [4]F32Vec2{
		{rect.X, rect.Y},                   //0
		{rect.X, rect.Y + rect.H},          //1
		{rect.X + rect.W, rect.Y + rect.H}, //2
		{rect.X + rect.W, rect.Y},          //3
	}
}

func orthoProjection(top, left, right, bottom, near, far float32) [16]float32 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	return [16]float32{
		float32(2. / rml), 0, 0, 0, // 1
		0, float32(2. / tmb), 0, 0, // 2
		0, 0, float32(-2. / fmn), 0, // 3
		float32(-(right + left) / rml), // 4
		float32(-(top + bottom) / tmb),
		float32(-(far + near) / fmn), 1}
}

type Animation struct {
	current  F32Vec2
	target   F32Vec2
	lifeTime float32
}

const ANIM_FINISH_TOLERANCE = 0.1

// Lifetime is the life of the animation. Animation speed is depends of the
// delta time and lifetime. For lifeTime parameter, 1.0 value is 1 seconds
func CreateAnimation(from, to F32Vec2, lifeTime float32) Animation {
	return Animation{
		current:  from,
		target:   to,
		lifeTime: lifeTime,
	}
}

// Returns current position of animation as x and y position
// If animation is finished, returned bool value will be true
func (anim *Animation) GetCurrentStep(deltaTime float32) (F32Vec2, bool) {
	if anim.lifeTime > 0 && deltaTime > 0 {
		anim.current.X += (anim.target.X - anim.current.X) / (anim.lifeTime / deltaTime)
		anim.current.Y += (anim.target.Y - anim.current.Y) / (anim.lifeTime / deltaTime)
		finishedX := math.Abs(float64(anim.target.X-anim.current.X)) < ANIM_FINISH_TOLERANCE
		finishedY := math.Abs(float64(anim.target.Y-anim.current.Y)) < ANIM_FINISH_TOLERANCE
		return anim.current, finishedX && finishedY
	}
	return anim.target, true
}

type AtomicBool struct {
	value int32
}

func (atomicBool *AtomicBool) Set(value bool) {
	var val int32 = 0
	if value == true {
		val = 1
	}
	atomic.StoreInt32(&atomicBool.value, val)
}

func (atomicBool *AtomicBool) Get() bool {
	val := atomic.LoadInt32(&atomicBool.value)
	if val == 0 {
		return false
	}
	return true
}
