package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type U8Color struct {
	R, G, B, A uint8
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

func (c U8Color) toF32() F32Color {
	return F32Color{
		R: float32(c.R) / 255,
		G: float32(c.G) / 255,
		B: float32(c.B) / 255,
		A: float32(c.A) / 255,
	}
}

type F32Color struct {
	R, G, B, A float32
}

type IntVec2 struct {
	X, Y int
}

func (v IntVec2) String() string {
	return fmt.Sprintf("(X: %d, Y: %d)", v.X, v.Y)
}

func (v IntVec2) toF32() F32Vec2 {
	return F32Vec2{X: float32(v.X), Y: float32(v.Y)}
}

func (v IntVec2) equals(v1 IntVec2) bool {
	return v.X == v1.X && v.Y == v1.Y
}

func (v IntVec2) inRect(rect IntRect) bool {
	return v.X >= rect.X && v.Y >= rect.Y && v.X < rect.X+rect.W && v.Y < rect.Y+rect.H
}

type F32Vec3 struct {
	X, Y, Z float32
}

func (v F32Vec3) toVec2() F32Vec2 {
	return F32Vec2{v.X, v.Y}
}

type F32Vec2 struct {
	X, Y float32
}

func (v F32Vec2) String() string {
	return fmt.Sprintf("(X: %f, Y: %f)", v.X, v.Y)
}

func (v F32Vec2) toInt() IntVec2 {
	x := int(math.Round(float64(v.X)))
	y := int(math.Round(float64(v.Y)))
	return IntVec2{x, y}
}

func (v F32Vec2) toVec3(Z float32) F32Vec3 {
	return F32Vec3{v.X, v.Y, Z}
}

func (v F32Vec2) plus(v2 F32Vec2) F32Vec2 {
	return F32Vec2{X: v.X + v2.X, Y: v.Y + v2.Y}
}

func (v F32Vec2) minus(v2 F32Vec2) F32Vec2 {
	return F32Vec2{X: v.X - v2.X, Y: v.Y - v2.Y}
}

func (v F32Vec2) divideS(S float32) F32Vec2 {
	return F32Vec2{X: v.X / S, Y: v.Y / S}
}

func (v F32Vec2) multiplyS(S float32) F32Vec2 {
	return F32Vec2{X: v.X * S, Y: v.Y * S}
}

func (v F32Vec2) length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}

func (v F32Vec2) normalized() F32Vec2 {
	return v.divideS(v.length())
}

func (v F32Vec2) perpendicular() F32Vec2 {
	return F32Vec2{X: v.Y, Y: -v.X}
}

func (v F32Vec2) isHorizontal() bool {
	return math.Abs(float64(v.X)) >= math.Abs(float64(v.Y))
}

type IntRect struct {
	X, Y, W, H int
}

func (rect IntRect) String() string {
	return fmt.Sprintf("(X: %d, Y: %d, W: %d, H: %d)", rect.X, rect.Y, rect.W, rect.H)
}

type F32Rect struct {
	X, Y, W, H float32
}

func (rect F32Rect) String() string {
	return fmt.Sprintf("(X: %f, Y: %f, W: %f, H: %f)", rect.X, rect.Y, rect.W, rect.H)
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

type Animation struct {
	current  F32Vec2
	target   F32Vec2
	lifeTime float32
	finished bool
}

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
	if anim.lifeTime > 0 && deltaTime > 0 && !anim.finished {
		anim.current.X += (anim.target.X - anim.current.X) / (anim.lifeTime / deltaTime)
		anim.current.Y += (anim.target.Y - anim.current.Y) / (anim.lifeTime / deltaTime)
		finishedX := math.Abs(float64(anim.target.X-anim.current.X)) < 0.1
		finishedY := math.Abs(float64(anim.target.Y-anim.current.Y)) < 0.1
		anim.finished = finishedX && finishedY
		return anim.current, anim.finished
	}
	return anim.target, true
}

type AtomicBool int32

func (atomicBool *AtomicBool) Set(value bool) {
	var val int32 = 0
	if value == true {
		val = 1
	}
	atomic.StoreInt32((*int32)(atomicBool), val)
}

func (atomicBool *AtomicBool) Get() bool {
	val := atomic.LoadInt32((*int32)(atomicBool))
	return val != 0
}

func (atomicBool *AtomicBool) WaitUntil(val bool) {
	for atomicBool.Get() != val {
		time.Sleep(time.Microsecond)
	}
}

func parseSizeString(size string) (int, int, bool) {
	// Size must be in form of '10x10'
	values := strings.Split(size, "x")
	if len(values) != 2 {
		return 0, 0, false
	}
	width, err := strconv.Atoi(values[0])
	if err != nil {
		return 0, 0, false
	}
	height, err := strconv.Atoi(values[1])
	if err != nil {
		return 0, 0, false
	}
	return width, height, true
}

func mergeStringArray(arr []string) string {
	str := ""
	for i := 0; i < len(arr)-1; i++ {
		str += arr[i] + " "
	}
	str += arr[len(arr)-1]
	return str
}

type BitMask uint32

func (mask BitMask) String() string {
	str := ""
	for i := 31; i >= 0; i-- {
		if mask.has(1 << i) {
			str += "1"
		} else {
			str += "0"
		}
	}
	return str
}

func (mask *BitMask) enable(flag BitMask) {
	*mask |= flag
}

func (mask *BitMask) disable(flag BitMask) {
	*mask &= ^flag
}

func (mask *BitMask) toggle(flag BitMask) {
	*mask ^= flag
}

// Enables flag if cond is true, disables otherwise.
func (mask *BitMask) enableif(flag BitMask, cond bool) {
	if cond {
		mask.enable(flag)
	} else {
		mask.disable(flag)
	}
}

func (mask BitMask) has(flag BitMask) bool {
	return mask&flag == flag
}

// Returns true if the mask only has the flag
func (mask BitMask) hasonly(flag BitMask) bool {
	return mask.has(flag) && mask|flag == flag
}
