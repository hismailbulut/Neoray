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
