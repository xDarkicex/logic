package fuzzy

import (
	"math"
)

// Triangular creates a triangular membership function.
// It peaks at b (value 1.0) and falls to 0.0 at a and c.
func Triangular(a, b, c float64) MembershipFunc {
	return func(x float64) TruthValue {
		if x <= a || x >= c {
			return 0.0
		}
		if x == b {
			return 1.0
		}
		if x < b {
			return TruthValue((x - a) / (b - a))
		}
		return TruthValue((c - x) / (c - b))
	}
}

// Trapezoidal creates a trapezoidal membership function.
// It is flat (1.0) between b and c, and falls to 0.0 at a and d.
func Trapezoidal(a, b, c, d float64) MembershipFunc {
	return func(x float64) TruthValue {
		if x <= a || x >= d {
			return 0.0
		}
		if x >= b && x <= c {
			return 1.0
		}
		if x < b {
			return TruthValue((x - a) / (b - a))
		}
		return TruthValue((d - x) / (d - c))
	}
}

// Gaussian creates a Gaussian membership function.
// It follows the curve e^(-(x-mean)^2 / (2*stddev^2)).
func Gaussian(mean, stddev float64) MembershipFunc {
	// Precompute denominator for performance
	twoVariance := 2.0 * stddev * stddev
	return func(x float64) TruthValue {
		if stddev == 0 {
			if x == mean {
				return 1.0
			}
			return 0.0
		}
		diff := x - mean
		return TruthValue(math.Exp(-(diff * diff) / twoVariance))
	}
}

// Bell creates a generalized Bell membership function.
// Formula: 1 / (1 + |(x-c)/a|^(2b))
func Bell(a, b, c float64) MembershipFunc {
	return func(x float64) TruthValue {
		if a == 0 {
			if x == c {
				return 1.0
			}
			return 0.0
		}
		absDiff := math.Abs((x - c) / a)
		return TruthValue(1.0 / (1.0 + math.Pow(absDiff, 2.0*b)))
	}
}

// Sigmoid creates a sigmoidal membership function.
// Formula: 1 / (1 + e^(-a*(x-c)))
func Sigmoid(a, c float64) MembershipFunc {
	return func(x float64) TruthValue {
		return TruthValue(1.0 / (1.0 + math.Exp(-a*(x-c))))
	}
}

// Singleton creates a singleton membership function.
// It returns 1.0 exactly at value, and 0.0 everywhere else.
func Singleton(value float64) MembershipFunc {
	return func(x float64) TruthValue {
		if x == value {
			return 1.0
		}
		return 0.0
	}
}

// Laplace creates a Laplace-distribution-based membership function.
// Formula: e^(-|x-loc|/scale). BCI/EEG validated (iFuzzyAffectDuo 2025).
func Laplace(loc, scale float64) MembershipFunc {
	return func(x float64) TruthValue {
		if scale == 0 {
			if x == loc {
				return 1.0
			}
			return 0.0
		}
		return TruthValue(math.Exp(-math.Abs(x-loc) / scale))
	}
}
