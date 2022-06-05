package main

import (
	"sync/atomic"
)

// Animation used for calculating positions of animated objects (points)
type Animation struct {
	current  Vector2[float32]
	target   Vector2[float32]
	lifeTime float32
	finished bool
}

// Lifetime is the life of the animation. Animation speed is depends of the
// delta time and lifetime. For lifeTime parameter, 1.0 value is 1 seconds
func CreateAnimation(from, to Vector2[float32], lifeTime float32) Animation {
	return Animation{
		current:  from,
		target:   to,
		lifeTime: lifeTime,
	}
}

// Returns current position of animation
// If animation is finished, returned bool value will be true
func (anim *Animation) GetCurrentStep(deltaTime float32) (Vector2[float32], bool) {
	if anim.lifeTime > 0 && deltaTime > 0 && !anim.finished {
		anim.current = anim.current.plus(anim.target.minus(anim.current).divideS(anim.lifeTime / deltaTime))
		anim.finished = anim.target.distance(anim.current) < 0.1
		return anim.current, anim.finished
	}
	return anim.target, true
}

// Threadsafe bool value, use it's methods
type AtomicBool int32

func (atomicBool *AtomicBool) Set(value bool) {
	val := int32(0)
	if value == true {
		val = 1
	}
	atomic.StoreInt32((*int32)(atomicBool), val)
}

func (atomicBool *AtomicBool) Get() bool {
	val := atomic.LoadInt32((*int32)(atomicBool))
	return val != 0
}

// This is just for making code more readable
// Can hold up to 16 enums
type BitMask uint16

func (mask BitMask) String() string {
	str := ""
	for i := 15; i >= 0; i-- {
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
