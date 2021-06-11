package main

import "math"

type Animation struct {
	current  F32Vec2
	target   F32Vec2
	lifeTime float32
}

const AnimationFinishTolerance = 0.1

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
// If animation is finishe, returned bool value will be true
func (anim *Animation) GetCurrentStep(deltaTime float32) (F32Vec2, bool) {
	anim.current.X += (anim.target.X - anim.current.X) / (anim.lifeTime / deltaTime)
	anim.current.Y += (anim.target.Y - anim.current.Y) / (anim.lifeTime / deltaTime)
	finishedX := math.Abs(float64(anim.target.X-anim.current.X)) < AnimationFinishTolerance
	finishedY := math.Abs(float64(anim.target.Y-anim.current.Y)) < AnimationFinishTolerance
	return anim.current, finishedX && finishedY
}
