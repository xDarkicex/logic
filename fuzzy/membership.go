package fuzzy

import (
	"math"

	"github.com/xDarkicex/memory"
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

// Rectangle creates a rectangular membership function.
// Returns 1.0 for x in [start, end], 0.0 otherwise.
// Ported from fuzzylite's Rectangle term.
func Rectangle(start, end float64) MembershipFunc {
	return func(x float64) TruthValue {
		if x >= start && x <= end {
			return 1.0
		}
		return 0.0
	}
}

// Ramp creates a ramp membership function that rises from 0 at start to 1 at end.
// Returns 0 for x <= start, 1 for x >= end, linear in between.
// Ported from fuzzylite's Ramp term.
func Ramp(start, end float64) MembershipFunc {
	return func(x float64) TruthValue {
		if x <= start {
			return 0.0
		}
		if x >= end {
			return 1.0
		}
		return TruthValue((x - start) / (end - start))
	}
}

// SShape creates an S-shaped growth membership function over [start, end].
// Smoothly transitions from 0 to 1 using a quadratic spline.
// Ported from fuzzylite's SShape term.
func SShape(start, end float64) MembershipFunc {
	mid := (start + end) / 2.0
	invRange := 1.0 / (end - start)
	return func(x float64) TruthValue {
		if x <= start {
			return 0.0
		}
		if x >= end {
			return 1.0
		}
		if x <= mid {
			t := (x - start) * invRange
			return TruthValue(2.0 * t * t)
		}
		t := (end - x) * invRange
		return TruthValue(1.0 - 2.0*t*t)
	}
}

// ZShape creates a Z-shaped decay membership function over [start, end].
// Smoothly transitions from 1 to 0 using a quadratic spline.
// Ported from fuzzylite's ZShape term.
func ZShape(start, end float64) MembershipFunc {
	mid := (start + end) / 2.0
	invRange := 1.0 / (end - start)
	return func(x float64) TruthValue {
		if x <= start {
			return 1.0
		}
		if x >= end {
			return 0.0
		}
		if x <= mid {
			t := (x - start) * invRange
			return TruthValue(1.0 - 2.0*t*t)
		}
		t := (end - x) * invRange
		return TruthValue(2.0 * t * t)
	}
}

// PiShape creates a Pi-shaped membership function.
// Product of SShape(start, mid) * ZShape(mid, end), forming a smooth bump.
// Ported from fuzzylite's PiShape term.
func PiShape(bottom, top, start, end float64) MembershipFunc {
	s := SShape(bottom, top)
	z := ZShape(start, end)
	return func(x float64) TruthValue {
		return s(x) * z(x)
	}
}

// Cosine creates a cosine-based membership function over [center, center+width].
// Smooth, continuously differentiable transition from 1 to 0.
// Ported from fuzzylite's Cosine term.
func Cosine(center, width float64) MembershipFunc {
	return func(x float64) TruthValue {
		if x < center {
			return 0.0
		}
		if x > center+width {
			return 0.0
		}
		return TruthValue(0.5 * (1.0 + math.Cos(math.Pi*(x-center)/width)))
	}
}

// GaussianProduct creates an asymmetric Gaussian membership function.
// Uses sigmaLeft for x < mean, sigmaRight for x >= mean.
// Ported from fuzzylite's GaussianProduct term.
func GaussianProduct(mean, sigmaLeft, sigmaRight float64) MembershipFunc {
	twoLeft := 2.0 * sigmaLeft * sigmaLeft
	twoRight := 2.0 * sigmaRight * sigmaRight
	return func(x float64) TruthValue {
		diff := x - mean
		if x < mean {
			if sigmaLeft == 0 {
				return 0.0
			}
			return TruthValue(math.Exp(-(diff * diff) / twoLeft))
		}
		if sigmaRight == 0 {
			if x == mean {
				return 1.0
			}
			return 0.0
		}
		return TruthValue(math.Exp(-(diff * diff) / twoRight))
	}
}

// Spike creates a spike membership function.
// Returns 1 at center, decays as exp(-|x-center|/width).
// Ported from fuzzylite's Spike term.
func Spike(center, width float64) MembershipFunc {
	return func(x float64) TruthValue {
		if width == 0 {
			if x == center {
				return 1.0
			}
			return 0.0
		}
		return TruthValue(math.Exp(-math.Abs(x-center) / width))
	}
}

// Discrete creates a discrete membership function from x,y value pairs.
// The pairs slice is [x1, y1, x2, y2, ...] and must be sorted by x ascending.
// Uses linear interpolation between points. A Pool-backed copy is allocated.
// Ported from fuzzylite's Discrete term.
func Discrete(xyPairs []float64, pool *memory.Pool) MembershipFunc {
	n := len(xyPairs)
	data := memory.MustPoolSlice[float64](pool, n)
	data = data[:n]
	copy(data, xyPairs)
	count := n / 2
	return func(x float64) TruthValue {
		if x < data[0] {
			return TruthValue(data[1])
		}
		if x > data[n-2] {
			return TruthValue(data[n-1])
		}
		for i := 0; i < count-1; i++ {
			xi := data[2*i]
			yi := data[2*i+1]
			xj := data[2*i+2]
			yj := data[2*i+3]
			if x >= xi && x <= xj {
				if xi == xj {
					return TruthValue(yi)
				}
				t := (x - xi) / (xj - xi)
				return TruthValue(yi + t*(yj-yi))
			}
		}
		return 0.0
	}
}

// Binary creates a binary threshold membership function.
// Returns 1.0 if x meets the direction test against the inflection point.
// If direction is true, returns 1.0 for x >= inflection. If false, for x <= inflection.
// Ported from fuzzylite's Binary term.
func Binary(inflection float64, direction bool) MembershipFunc {
	return func(x float64) TruthValue {
		if direction {
			if x >= inflection {
				return 1.0
			}
			return 0.0
		}
		if x <= inflection {
			return 1.0
		}
		return 0.0
	}
}

// Concave creates a concave membership function that rises from 0 at start to 1 at end.
// The curve grows fast early then slows — useful for diminishing returns in recall strength.
// Ported from fuzzylite's Concave term.
func Concave(start, end float64) MembershipFunc {
	invRange := 1.0 / (end - start)
	return func(x float64) TruthValue {
		if x <= start {
			return 0.0
		}
		if x >= end {
			return 1.0
		}
		t := (x - start) * invRange
		return TruthValue(t / (1.0 + t))
	}
}

// SigmoidDifference creates a membership function from the difference of two sigmoids.
// Produces a smooth bump shape. Ported from fuzzylite's SigmoidDifference term.
func SigmoidDifference(left, leftInflection, right, rightInflection float64) MembershipFunc {
	return func(x float64) TruthValue {
		leftVal := 1.0 / (1.0 + math.Exp(-left*(x-leftInflection)))
		rightVal := 1.0 / (1.0 + math.Exp(-right*(x-rightInflection)))
		diff := leftVal - rightVal
		if diff < 0 {
			return 0.0
		}
		return TruthValue(diff)
	}
}

// SigmoidProduct creates a membership function from the product of two sigmoids.
// Produces a smooth bump shape (symmetric variant). Ported from fuzzylite's SigmoidProduct term.
func SigmoidProduct(left, leftInflection, right, rightInflection float64) MembershipFunc {
	return func(x float64) TruthValue {
		leftVal := 1.0 / (1.0 + math.Exp(-left*(x-leftInflection)))
		rightVal := 1.0 / (1.0 + math.Exp(-right*(x-rightInflection)))
		return TruthValue(leftVal * rightVal)
	}
}
