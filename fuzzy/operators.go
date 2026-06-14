package fuzzy

// TNorms (Fuzzy AND)

// MinTNorm returns the minimum of two TruthValues (Gödel t-norm).
func MinTNorm(a, b TruthValue) TruthValue {
	if a < b {
		return a
	}
	return b
}

// ProductTNorm returns the product of two TruthValues.
func ProductTNorm(a, b TruthValue) TruthValue {
	return a * b
}

// LukasiewiczTNorm returns max(0, a+b-1).
func LukasiewiczTNorm(a, b TruthValue) TruthValue {
	sum := a + b - 1.0
	if sum > 0.0 {
		return sum
	}
	return 0.0
}

// MinTNormVariadic returns the minimum of multiple TruthValues.
func MinTNormVariadic(values ...TruthValue) TruthValue {
	if len(values) == 0 {
		return 1.0 // Identity for AND
	}
	min := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] < min {
			min = values[i]
		}
	}
	return min
}

// TConorms (Fuzzy OR)

// MaxTConorm returns the maximum of two TruthValues.
func MaxTConorm(a, b TruthValue) TruthValue {
	if a > b {
		return a
	}
	return b
}

// ProbabilisticTConorm returns a + b - a*b.
func ProbabilisticTConorm(a, b TruthValue) TruthValue {
	return a + b - (a * b)
}

// LukasiewiczTConorm returns min(1, a+b).
func LukasiewiczTConorm(a, b TruthValue) TruthValue {
	sum := a + b
	if sum < 1.0 {
		return sum
	}
	return 1.0
}

// MaxTConormVariadic returns the maximum of multiple TruthValues.
func MaxTConormVariadic(values ...TruthValue) TruthValue {
	if len(values) == 0 {
		return 0.0 // Identity for OR
	}
	max := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] > max {
			max = values[i]
		}
	}
	return max
}

// Negation

// StandardNegation returns 1 - a.
func StandardNegation(a TruthValue) TruthValue {
	return 1.0 - a
}

// Implications

// GodelImplication returns 1 if a <= b, else b.
func GodelImplication(a, b TruthValue) TruthValue {
	if a <= b {
		return 1.0
	}
	return b
}

// GoguenImplication returns 1 if a == 0, else min(1, b/a).
func GoguenImplication(a, b TruthValue) TruthValue {
	if a == 0.0 {
		return 1.0
	}
	res := b / a
	if res < 1.0 {
		return res
	}
	return 1.0
}

// LukasiewiczImplication returns min(1, 1-a+b).
func LukasiewiczImplication(a, b TruthValue) TruthValue {
	res := 1.0 - a + b
	if res < 1.0 {
		return res
	}
	return 1.0
}

// KleeneDienesImplication returns max(1-a, b).
func KleeneDienesImplication(a, b TruthValue) TruthValue {
	return MaxTConorm(StandardNegation(a), b)
}

// TNorms (additional from fuzzylite)

// EinsteinProduct returns the Einstein product t-norm: (a*b) / (2 - (a+b-a*b)).
func EinsteinProduct(a, b TruthValue) TruthValue {
	return (a * b) / (2.0 - (a + b - a*b))
}

// HamacherProduct returns the Hamacher product t-norm: (a*b) / (a+b-a*b).
// Returns 0 if both inputs are 0.
func HamacherProduct(a, b TruthValue) TruthValue {
	if a == 0.0 && b == 0.0 {
		return 0.0
	}
	return (a * b) / (a + b - a*b)
}

// NilpotentMinimum returns the nilpotent minimum t-norm.
// Returns min(a,b) if a+b > 1, else 0.
func NilpotentMinimum(a, b TruthValue) TruthValue {
	if a+b > 1.0 {
		return MinTNorm(a, b)
	}
	return 0.0
}

// DrasticProduct returns the drastic product t-norm.
// Returns min(a,b) if max(a,b) == 1, else 0.
func DrasticProduct(a, b TruthValue) TruthValue {
	if a == 1.0 || b == 1.0 {
		return MinTNorm(a, b)
	}
	return 0.0
}

// TConorms (additional from fuzzylite)

// EinsteinSum returns the Einstein sum t-conorm: (a+b) / (1 + a*b).
func EinsteinSum(a, b TruthValue) TruthValue {
	return (a + b) / (1.0 + a*b)
}

// HamacherSum returns the Hamacher sum t-conorm: (a+b-2ab) / (1-ab).
// Returns 1 if a*b == 1.
func HamacherSum(a, b TruthValue) TruthValue {
	if a*b == 1.0 {
		return 1.0
	}
	return (a + b - 2.0*a*b) / (1.0 - a*b)
}

// NilpotentMaximum returns the nilpotent maximum t-conorm.
// Returns max(a,b) if a+b < 1, else 1.
func NilpotentMaximum(a, b TruthValue) TruthValue {
	if a+b < 1.0 {
		return MaxTConorm(a, b)
	}
	return 1.0
}

// DrasticSum returns the drastic sum t-conorm.
// Returns max(a,b) if min(a,b) == 0, else 1.
func DrasticSum(a, b TruthValue) TruthValue {
	if a == 0.0 || b == 0.0 {
		return MaxTConorm(a, b)
	}
	return 1.0
}

// NormalizedSum returns the normalized sum t-conorm: (a+b) / max(1, a+b).
func NormalizedSum(a, b TruthValue) TruthValue {
	s := a + b
	if s > 1.0 {
		return 1.0
	}
	return s
}

// UnboundedSum returns the unbounded sum t-conorm: a + b.
func UnboundedSum(a, b TruthValue) TruthValue {
	return a + b
}
