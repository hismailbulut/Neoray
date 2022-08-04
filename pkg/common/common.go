package common

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

type Numbers interface {
	Integers | UnsignedIntegers | Floats
}

func Min[T Numbers](v1, v2 T) T {
	if v1 < v2 {
		return v1
	}
	return v2
}

func Max[T Numbers](v1, v2 T) T {
	if v1 > v2 {
		return v1
	}
	return v2
}

func Clamp[T Numbers](v, minv, maxv T) T {
	return Min(maxv, Max(minv, v))
}

func Abs[T Numbers](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

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
	return anim.from.Add(anim.to.Sub(anim.from).DivS(anim.lifeTime).MulS(Min(anim.time, anim.lifeTime)))
}

func (anim *Animation) IsFinished() bool {
	return anim.time >= anim.lifeTime
}

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
