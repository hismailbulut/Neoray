package main

type Animation struct {
	current  f32vec2
	target   f32vec2
	lifeTime float32
}

// Lifetime is the life of the animation. Animation speed is depends of the
// delta time and lifetime. For lifeTime parameter, 1.0 value is 1 seconds
func CreateAnimation(from, target f32vec2, lifeTime float32) Animation {
	return Animation{
		current:  from,
		target:   target,
		lifeTime: lifeTime,
	}
}

// Returns current position of animation as x and y position
// If the animation finished, the last bool value will set to true
func (anim *Animation) GetCurrentStep(delta float32) (f32vec2, bool) {
	anim.current.X += (anim.target.X - anim.current.X) / (anim.lifeTime / delta)
	anim.current.Y += (anim.target.Y - anim.current.Y) / (anim.lifeTime / delta)
	return anim.current, anim.current == anim.target
}
