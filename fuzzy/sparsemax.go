package fuzzy

import (
	"sort"

	"github.com/xDarkicex/memory"
)

// Sparsemax computes the sparse softmax projection of src onto the probability simplex.
// Unlike softmax, sparsemax produces outputs where many entries are exactly zero.
// Algorithm from Martins & Astudillo (2016): "From Softmax to Sparsemax: A Sparse Model of Attention."
// Extracted from gorgonia's op_sparsemax.go.
//
// If dst is nil, a Pool-backed slice is allocated. Returns dst.
// Time: O(n log n) due to sorting, Space: O(n).
func Sparsemax(src []float64, dst []float64, pool *memory.Pool) []float64 {
	n := len(src)
	sorted := sparsemaxSortCopy(src, pool)
	threshold := sparsemaxFindThreshold(sorted)
	if dst == nil {
		dst = memory.MustPoolSlice[float64](pool, n)
		dst = dst[:n]
	}
	sparsemaxApply(src, threshold, dst)
	return dst
}

// sparsemaxSortCopy returns a Pool-backed copy of src sorted in descending order.
// CC=2.
func sparsemaxSortCopy(src []float64, pool *memory.Pool) []float64 {
	n := len(src)
	sorted := memory.MustPoolSlice[float64](pool, n)
	sorted = sorted[:n]
	copy(sorted, src)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] > sorted[j]
	})
	return sorted
}

// sparsemaxFindThreshold finds the threshold τ(z) for the sparsemax projection.
// Given sorted z₁ ≥ z₂ ≥ ... ≥ zₙ, finds k = max{j | 1 + j·zⱼ > Σᵢ₌₁ʲ zᵢ}
// and returns τ = (Σᵢ₌₁ᵏ zᵢ - 1) / k.
// CC=6.
func sparsemaxFindThreshold(sorted []float64) float64 {
	var cumSum, prevCum float64
	maxIndex := 0

	for j, z := range sorted {
		k := 1 + float64(j+1)*z
		prevCum += z
		if k > prevCum {
			maxIndex = j + 1
			cumSum += z
		}
	}

	if maxIndex == 0 {
		return 0
	}
	return (cumSum - 1.0) / float64(maxIndex)
}

// sparsemaxApply applies the threshold: out[i] = max(src[i] - threshold, 0).
// CC=2.
func sparsemaxApply(src []float64, threshold float64, dst []float64) {
	for i, v := range src {
		if v > threshold {
			dst[i] = v - threshold
		} else {
			dst[i] = 0
		}
	}
}
