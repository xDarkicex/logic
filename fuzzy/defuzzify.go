package fuzzy

// Centroid returns the center of gravity of the fuzzy set.
// Formula: ∫x·μ(x)dx / ∫μ(x)dx
func Centroid(set *FuzzySet) float64 {
	var num, den float64
	for i, x := range set.Universe {
		m := float64(set.Members[i])
		num += x * m
		den += m
	}
	if den == 0 {
		return 0 // Avoid division by zero
	}
	return num / den
}

// MeanOfMax returns the average of the x values where μ(x) is maximized.
func MeanOfMax(set *FuzzySet) float64 {
	var maxM TruthValue
	var sum float64
	var count int

	for i, m := range set.Members {
		if m > maxM {
			maxM = m
			sum = set.Universe[i]
			count = 1
		} else if m == maxM && maxM > 0 {
			sum += set.Universe[i]
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// SmallestOfMax returns the smallest x value where μ(x) is maximized.
func SmallestOfMax(set *FuzzySet) float64 {
	var maxM TruthValue
	var smallestX float64
	first := true

	for i, m := range set.Members {
		if m > maxM || (m == maxM && first && maxM > 0) {
			maxM = m
			smallestX = set.Universe[i]
			first = false
		}
	}
	return smallestX
}

// LargestOfMax returns the largest x value where μ(x) is maximized.
func LargestOfMax(set *FuzzySet) float64 {
	var maxM TruthValue
	var largestX float64
	first := true

	for i, m := range set.Members {
		// Use >= to continually update largestX when m == maxM
		if m > maxM || (m == maxM && maxM > 0) || (first && maxM == 0) {
			maxM = m
			largestX = set.Universe[i]
			first = false
		}
	}
	return largestX
}

// Bisector returns the x value that divides the area under the curve into two equal halves.
func Bisector(set *FuzzySet) float64 {
	var totalArea float64
	for _, m := range set.Members {
		totalArea += float64(m)
	}
	
	if totalArea == 0 {
		return 0
	}

	targetArea := totalArea / 2.0
	var currentArea float64

	for i, m := range set.Members {
		currentArea += float64(m)
		if currentArea >= targetArea {
			return set.Universe[i]
		}
	}
	
	// Fallback to the last element
	if len(set.Universe) > 0 {
		return set.Universe[len(set.Universe)-1]
	}
	return 0
}

// WeightedAverageDefuzz computes the weighted average of crisp values.
// Used primarily for standalone evaluations, TSK uses this internally.
func WeightedAverageDefuzz(values, weights []float64) float64 {
	var num, den float64
	length := len(values)
	if len(weights) < length {
		length = len(weights)
	}

	for i := 0; i < length; i++ {
		num += values[i] * weights[i]
		den += weights[i]
	}

	if den == 0 {
		return 0
	}
	return num / den
}
