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
