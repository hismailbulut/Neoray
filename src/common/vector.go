package common

import (
	"fmt"
	"math"
)

// Zero values for reducing allocations
var (
	ZeroVector2F32 = Vector2[float32]{}
	ZeroVector2INT = Vector2[int]{}
)

// We will use generics for vectors

type Vector2[T SignedNumbers] struct {
	X, Y T
}

// just a shortcut
func Vec2[T SignedNumbers](X, Y T) Vector2[T] {
	return Vector2[T]{X: X, Y: Y}
}

func (v Vector2[T]) Width() T {
	return v.X
}

func (v Vector2[T]) Height() T {
	return v.Y
}

var (
	Vec2Up    = Vector2[float32]{X: 0, Y: -1}
	Vec2Down  = Vector2[float32]{X: 0, Y: 1}
	Vec2Left  = Vector2[float32]{X: -1, Y: 0}
	Vec2Right = Vector2[float32]{X: 1, Y: 0}
)

func (v Vector2[T]) String() string {
	return fmt.Sprintf("Vec2(X: %v, Y: %v)", v.X, v.Y)
}

func (v Vector2[T]) ToVec3(Z T) Vector3[T] {
	return Vector3[T]{X: v.X, Y: v.Y, Z: Z}
}

func (v Vector2[T]) ToInt() Vector2[int] {
	return Vector2[int]{
		X: int(math.Floor(float64(v.X))),
		Y: int(math.Floor(float64(v.Y))),
	}
}

func (v Vector2[T]) Plus(v1 Vector2[T]) Vector2[T] {
	return Vector2[T]{X: v.X + v1.X, Y: v.Y + v1.Y}
}

func (v Vector2[T]) Minus(v1 Vector2[T]) Vector2[T] {
	return Vector2[T]{X: v.X - v1.X, Y: v.Y - v1.Y}
}

func (v Vector2[T]) Multiply(v1 Vector2[T]) Vector2[T] {
	return Vector2[T]{X: v.X * v1.X, Y: v.Y * v1.Y}
}

func (v Vector2[T]) Equals(v1 Vector2[T]) bool {
	return v.X == v1.X && v.Y == v1.Y
}

func (v Vector2[T]) DivideScalar(S T) Vector2[T] {
	return Vector2[T]{X: v.X / S, Y: v.Y / S}
}

func (v Vector2[T]) MultiplyScalar(S T) Vector2[T] {
	return Vector2[T]{X: v.X * S, Y: v.Y * S}
}

func (v Vector2[T]) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}

func (v Vector2[T]) LengthSquare() float32 {
	return float32(v.X*v.X + v.Y*v.Y)
}

func (v Vector2[T]) Distance(v2 Vector2[T]) float32 {
	return v2.Minus(v).Length()
}

func (v Vector2[T]) DistanceSquare(v2 Vector2[T]) float32 {
	return v2.Minus(v).LengthSquare()
}

func (v Vector2[T]) Normalized() Vector2[T] {
	return v.DivideScalar(T(v.Length()))
}

func (v Vector2[T]) Perp() Vector2[T] {
	return Vector2[T]{X: v.Y, Y: -v.X}
}

func (v Vector2[T]) IsHorizontal() bool {
	return math.Abs(float64(v.X)) >= math.Abs(float64(v.Y))
}

func (v Vector2[T]) IsInRect(rect Rectangle[T]) bool {
	return v.X >= rect.X && v.Y >= rect.Y && v.X < rect.X+rect.W && v.Y < rect.Y+rect.H
}

type Vector3[T SignedNumbers] struct {
	X, Y, Z T
}

func Vec3[T SignedNumbers](X, Y, Z T) Vector3[T] {
	return Vector3[T]{X: X, Y: Y, Z: Z}
}

func (v Vector3[T]) ToVec2() Vector2[T] {
	return Vector2[T]{X: v.X, Y: v.Y}
}
