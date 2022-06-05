package main

import (
	"fmt"
	"math"
)

// Constraints
type Integers interface {
	int | int8 | int16 | int32 | int64
}

type UnsignedIntegers interface {
	uint | uint8 | uint16 | uint32 | uint64
}

type Floats interface {
	float32 | float64
}

type SignedNumbers interface {
	Integers | Floats
}

type Numbers interface {
	SignedNumbers | UnsignedIntegers
}

// We can't use generics for color types

type F32Color struct {
	R, G, B, A float32
}

type U8Color struct {
	R, G, B, A uint8
}

func (color U8Color) pack() uint32 {
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

// We will use generics for vectors

type Vector2[T SignedNumbers] struct {
	X, Y T
}

var (
	Vec2Up    = Vector2[float32]{X: 0, Y: -1}
	Vec2Down  = Vector2[float32]{X: 0, Y: 1}
	Vec2Left  = Vector2[float32]{X: -1, Y: 0}
	Vec2Right = Vector2[float32]{X: 1, Y: 0}
)

func (v Vector2[T]) String() string {
	return fmt.Sprint("(X: ", v.X, ", Y: ", v.Y, ")", v.X, v.Y)
}

func (v Vector2[T]) toF32() Vector2[float32] {
	return Vector2[float32]{X: float32(v.X), Y: float32(v.Y)}
}

func (v Vector2[T]) toVec3(Z T) Vector3[T] {
	return Vector3[T]{X: v.X, Y: v.Y, Z: Z}
}

func (v Vector2[T]) toInt() Vector2[int] {
	return Vector2[int]{
		X: int(math.Floor(float64(v.X))),
		Y: int(math.Floor(float64(v.Y))),
	}
}

func (v Vector2[T]) plus(v1 Vector2[T]) Vector2[T] {
	return Vector2[T]{X: v.X + v1.X, Y: v.Y + v1.Y}
}

func (v Vector2[T]) minus(v1 Vector2[T]) Vector2[T] {
	return Vector2[T]{X: v.X - v1.X, Y: v.Y - v1.Y}
}

func (v Vector2[T]) equals(v1 Vector2[T]) bool {
	return v.X == v1.X && v.Y == v1.Y
}

func (v Vector2[T]) divideS(S T) Vector2[T] {
	return Vector2[T]{X: v.X / S, Y: v.Y / S}
}

func (v Vector2[T]) multiplyS(S T) Vector2[T] {
	return Vector2[T]{X: v.X * S, Y: v.Y * S}
}

func (v Vector2[T]) length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}

func (v Vector2[T]) distance(v2 Vector2[T]) float32 {
	return v2.minus(v).length()
}

func (v Vector2[T]) normalized() Vector2[T] {
	return v.divideS(T(v.length()))
}

func (v Vector2[T]) perpendicular() Vector2[T] {
	return Vector2[T]{X: v.Y, Y: -v.X}
}

func (v Vector2[T]) isHorizontal() bool {
	return math.Abs(float64(v.X)) >= math.Abs(float64(v.Y))
}

func (v Vector2[T]) inRect(rect Rectangle[T]) bool {
	return v.X >= rect.X && v.Y >= rect.Y && v.X < rect.X+rect.W && v.Y < rect.Y+rect.H
}

type Vector3[T SignedNumbers] struct {
	X, Y, Z T
}

func (v Vector3[T]) toVec2() Vector2[T] {
	return Vector2[T]{X: v.X, Y: v.Y}
}

type Rectangle[T SignedNumbers] struct {
	X, Y, W, H T
}

func (rect Rectangle[T]) String() string {
	return fmt.Sprint("(X: ", rect.X, ", Y: ", rect.Y, ", W: ", rect.W, ", H: ", rect.H, ")")
}

func (rect Rectangle[T]) toInt() Rectangle[int] {
	return Rectangle[int]{
		X: int(math.Floor(float64(rect.X))),
		Y: int(math.Floor(float64(rect.Y))),
		W: int(math.Floor(float64(rect.W))),
		H: int(math.Floor(float64(rect.H))),
	}
}

func min[T Numbers](v1, v2 T) T {
	if v1 < v2 {
		return v1
	}
	return v2
}

func max[T Numbers](v1, v2 T) T {
	if v1 > v2 {
		return v1
	}
	return v2
}

func clamp[T Numbers](v, minv, maxv T) T {
	return min(maxv, max(minv, v))
}

func abs[T Numbers](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

func ortho(top, left, right, bottom, near, far float32) [16]float32 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	matrix := [16]float32{}
	matrix[0] = 2 / rml
	matrix[5] = 2 / tmb
	matrix[10] = -2 / fmn
	matrix[12] = -(right + left) / rml
	matrix[13] = -(top + bottom) / tmb
	matrix[14] = -(far + near) / fmn
	matrix[15] = 1
	return matrix
}
