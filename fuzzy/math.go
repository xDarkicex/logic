package fuzzy

import (
	"math"

	"github.com/xDarkicex/memory"
)

// Softplus computes the numerically stable softplus function: log(1 + exp(x)).
// Returns 0 for x < -708, x for x > 16, and log1p(exp(x)) otherwise.
// Extracted from gorgonia's math.go — softplus is the stable form of log1p(exp(x)).
func Softplus(x float64) float64 {
	if x < -708 {
		return 0
	}
	if x > 16 {
		return x
	}
	return math.Log1p(math.Exp(x))
}

// Softmax computes the numerically stable softmax of src, writing to dst.
// If dst is nil, a Pool-backed slice is allocated. Returns dst.
// Time: O(n), Space: O(n).
func Softmax(src []float64, dst []float64, pool *memory.Pool) []float64 {
	n := len(src)
	if dst == nil {
		dst = memory.MustPoolSlice[float64](pool, n)
		dst = dst[:n]
	}

	// Subtract max for numerical stability
	maxVal := src[0]
	for i := 1; i < n; i++ {
		if src[i] > maxVal {
			maxVal = src[i]
		}
	}

	var sum float64
	for i, v := range src {
		dst[i] = math.Exp(v - maxVal)
		sum += dst[i]
	}

	for i := range dst {
		dst[i] /= sum
	}
	return dst
}

// LogSoftmax computes log(softmax(src)) in a numerically stable way, writing to dst.
// If dst is nil, a Pool-backed slice is allocated. Returns dst.
// Time: O(n), Space: O(n).
func LogSoftmax(src []float64, dst []float64, pool *memory.Pool) []float64 {
	n := len(src)
	if dst == nil {
		dst = memory.MustPoolSlice[float64](pool, n)
		dst = dst[:n]
	}

	maxVal := src[0]
	for i := 1; i < n; i++ {
		if src[i] > maxVal {
			maxVal = src[i]
		}
	}

	var sum float64
	for _, v := range src {
		sum += math.Exp(v - maxVal)
	}
	logSum := math.Log(sum)

	for i, v := range src {
		dst[i] = v - maxVal - logSum
	}
	return dst
}
