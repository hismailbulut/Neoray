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

type Vector2[T Numbers] struct {
	X, Y T
}

func (v Vector2[T]) String() string {
	return fmt.Sprintf("Vector2(X: %v, Y: %v)", v.X, v.Y)
}

// just a shortcut
func Vec2[T Numbers](X, Y T) Vector2[T] {
	return Vector2[T]{X: X, Y: Y}
}

func (v Vector2[T]) Width() T {
	return v.X
}

func (v Vector2[T]) Height() T {
	return v.Y
}

func (v Vector2[T]) ToInt() Vector2[int] {
	return Vector2[int]{
		X: int(math.Floor(float64(v.X))),
		Y: int(math.Floor(float64(v.Y))),
	}
}

// Add two vectors
func (v Vector2[T]) Add(v1 Vector2[T]) Vector2[T] {
	v.X += v1.X
	v.Y += v1.Y
	return v
}

// Subtract v1 from v
func (v Vector2[T]) Sub(v1 Vector2[T]) Vector2[T] {
	v.X -= v1.X
	v.Y -= v1.Y
	return v
}

// Dot product of two vectors
func (v Vector2[T]) Mul(v1 Vector2[T]) Vector2[T] {
	v.X *= v1.X
	v.Y *= v1.Y
	return v
}

// Multiplies the vector by a scalar
func (v Vector2[T]) MulS(S T) Vector2[T] {
	v.X *= S
	v.Y *= S
	return v
}

// Divides v by v1
func (v Vector2[T]) Div(v1 Vector2[T]) Vector2[T] {
	v.X /= v1.X
	v.Y /= v1.Y
	return v
}

// Divides the vector by a scalar
func (v Vector2[T]) DivS(S T) Vector2[T] {
	v.X /= S
	v.Y /= S
	return v
}

func (v Vector2[T]) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}

// Does not applies sqrt, faster
func (v Vector2[T]) LengthSquared() float32 {
	return float32(v.X*v.X + v.Y*v.Y)
}

func (v Vector2[T]) Distance(v2 Vector2[T]) float32 {
	return v2.Sub(v).Length()
}

// Does not applies sqrt, faster
func (v Vector2[T]) DistanceSquared(v2 Vector2[T]) float32 {
	return v2.Sub(v).LengthSquared()
}

func (v Vector2[T]) Normalized() Vector2[T] {
	return v.DivS(T(v.Length()))
}

// Returns the perpenicular vector of v
func (v Vector2[T]) Perpendicular() Vector2[T] {
	v.X, v.Y = v.Y, -v.X
	return v
}

// Returns true if the vectors are equal
func (v Vector2[T]) Equals(v1 Vector2[T]) bool {
	return v.X == v1.X && v.Y == v1.Y
}

// Returns true if the vector is horizontal
func (v Vector2[T]) IsHorizontal() bool {
	return Abs(v.X) >= Abs(v.Y)
}

// Returns true if the vector is in given rectangle
func (v Vector2[T]) IsInRect(rect Rectangle[T]) bool {
	return v.X >= rect.X && v.Y >= rect.Y && v.X < rect.X+rect.W && v.Y < rect.Y+rect.H
}
