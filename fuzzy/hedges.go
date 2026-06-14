package fuzzy

import "math"

// Very returns a membership function squared (concentration).
func Very(mf MembershipFunc) MembershipFunc {
	return func(x float64) TruthValue {
		v := mf(x)
		return v * v
	}
}

// Somewhat returns a membership function square-rooted (dilation).
func Somewhat(mf MembershipFunc) MembershipFunc {
	return func(x float64) TruthValue {
		return TruthValue(math.Sqrt(float64(mf(x))))
	}
}

// Slightly returns Very(Somewhat(mf)).
func Slightly(mf MembershipFunc) MembershipFunc {
	return Very(Somewhat(mf))
}

// Extremely returns a membership function cubed.
func Extremely(mf MembershipFunc) MembershipFunc {
	return func(x float64) TruthValue {
		v := mf(x)
		return v * v * v
	}
}

// Indeed returns a membership function intensifying contrast.
func Indeed(mf MembershipFunc) MembershipFunc {
	return func(x float64) TruthValue {
		v := float64(mf(x))
		if v <= 0.5 {
			return TruthValue(2.0 * v * v)
		}
		inv := 1.0 - v
		return TruthValue(1.0 - 2.0*inv*inv)
	}
}

// Not returns the complement of a membership function.
func Not(mf MembershipFunc) MembershipFunc {
	return func(x float64) TruthValue {
		return 1.0 - mf(x)
	}
}
