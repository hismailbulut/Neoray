package common

import "sync/atomic"

// Animation used for calculating positions of animated objects (points)
type Animation struct {
	from     Vector2[float32]
	to       Vector2[float32]
	time     float32
	lifeTime float32
}

// Lifetime is the life of the animation. Animation speed is depends of the
// delta time and lifetime. For lifeTime parameter, 1.0 value is 1 seconds
func NewAnimation(from, to Vector2[float32], lifeTime float32) Animation {
	return Animation{
		from:     from,
		to:       to,
		time:     0,
		lifeTime: lifeTime,
	}
}

// Returns current position of animation
func (anim *Animation) Step(delta float32) Vector2[float32] {
	if anim.lifeTime <= 0 {
		return anim.to
	}
	anim.time += delta
	return anim.from.Plus(anim.to.Minus(anim.from).DivideScalar(anim.lifeTime).MultiplyScalar(Min(anim.time, anim.lifeTime)))
}

func (anim *Animation) IsFinished() bool {
	return anim.time >= anim.lifeTime
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

// type AtomicInt int64
//
// func (atomicInt *AtomicInt) Set(value int64) {
//     atomic.StoreInt64((*int64)(atomicInt), value)
// }
//
// func (atomicInt *AtomicInt) Get() int64 {
//     return atomic.LoadInt64((*int64)(atomicInt))
// }
//
// func (atomicInt *AtomicInt) Increment() {
//     atomicInt.Set(atomicInt.Get() + 1)
// }

// This is just for making code more readable
// Can hold up to 32 enums
type BitMask uint32

func (mask BitMask) String() string {
	str := ""
	for i := 31; i >= 0; i-- {
		if mask.Has(1 << i) {
			str += "1"
		} else {
			str += "0"
		}
	}
	return str
}

func (mask *BitMask) Enable(flag BitMask) {
	*mask |= flag
}

func (mask *BitMask) Disable(flag BitMask) {
	*mask &= ^flag
}

func (mask *BitMask) Toggle(flag BitMask) {
	*mask ^= flag
}

// Enables flag if cond is true, disables otherwise.
func (mask *BitMask) EnableIf(flag BitMask, cond bool) {
	if cond {
		mask.Enable(flag)
	} else {
		mask.Disable(flag)
	}
}

func (mask BitMask) Has(flag BitMask) bool {
	return mask&flag == flag
}

// Returns true if the mask only has the flag
func (mask BitMask) HasOnly(flag BitMask) bool {
	return mask.Has(flag) && mask|flag == flag
}

func (mask *BitMask) Clear() {
	*mask = *mask & BitMask(0)
}
