package fuzzy

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func TestSparsemax(t *testing.T) {
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Reset()

	t.Run("normal case with dst provided", func(t *testing.T) {
		src := []float64{1, 2, 3}
		dst := memory.MustPoolSlice[float64](pool, 3)
		dst = dst[:3]
		Sparsemax(src, dst, pool)

		var sum float64
		zeros := 0
		for _, v := range dst {
			sum += v
			if v == 0 {
				zeros++
			}
		}
		if !approxEqual(sum, 1.0) {
			t.Errorf("sparsemax sum = %v, want 1.0", sum)
		}
		if zeros == 0 {
			t.Logf("no zeros produced for distinct input (expected for sparsemax)")
		}
	})

	t.Run("nil dst allocates", func(t *testing.T) {
		src := []float64{1, 2, 3}
		dst := Sparsemax(src, nil, pool)
		if len(dst) != 3 {
			t.Errorf("len = %d, want 3", len(dst))
		}
	})

	t.Run("single element returns 1", func(t *testing.T) {
		src := []float64{5}
		dst := Sparsemax(src, nil, pool)
		if !approxEqual(dst[0], 1.0) {
			t.Errorf("single element sparsemax = %v, want 1.0", dst[0])
		}
	})

	t.Run("all equal values are uniform", func(t *testing.T) {
		src := []float64{2, 2, 2, 2}
		dst := Sparsemax(src, nil, pool)
		var sum float64
		for _, v := range dst {
			sum += v
		}
		if !approxEqual(sum, 1.0) {
			t.Errorf("sum = %v, want 1.0", sum)
		}
		for _, v := range dst {
			if !approxEqual(v, 0.25) {
				t.Errorf("equal values sparsemax = %v, want 0.25", dst)
				break
			}
		}
	})

	t.Run("highly skewed input produces sparsity", func(t *testing.T) {
		src := []float64{10, 0.1, 0.1, 0.1}
		dst := Sparsemax(src, nil, pool)

		var sum float64
		zeros := 0
		for _, v := range dst {
			sum += v
			if v == 0 {
				zeros++
			}
		}

		if !approxEqual(sum, 1.0) {
			t.Errorf("sum = %v, want 1.0", sum)
		}
		if zeros < 1 {
			t.Errorf("sparsemax on skewed input should produce zeros, got %d zeros", zeros)
		}
	})

	t.Run("output is non-negative", func(t *testing.T) {
		src := []float64{-1, 0, 1}
		dst := Sparsemax(src, nil, pool)
		for i, v := range dst {
			if v < 0 {
				t.Errorf("sparsemax[%d] = %v, want non-negative", i, v)
			}
		}
	})

	t.Run("order preserved", func(t *testing.T) {
		src := []float64{1, 5, 3}
		dst := Sparsemax(src, nil, pool)
		for i := 0; i < len(src); i++ {
			for j := i + 1; j < len(src); j++ {
				if src[i] > src[j] && dst[i] < dst[j] {
					t.Errorf("order not preserved: src[%d]=%v > src[%d]=%v but dst[%d]=%v < dst[%d]=%v",
						i, src[i], j, src[j], i, dst[i], j, dst[j])
				}
			}
		}
	})
}

func TestSparsemaxFindThreshold(t *testing.T) {
	t.Run("largest element wins", func(t *testing.T) {
		sorted := []float64{5, 2, 1}
		threshold := sparsemaxFindThreshold(sorted)
		if threshold <= 0 {
			t.Errorf("threshold = %v, expected positive", threshold)
		}
		if threshold >= 5 {
			t.Errorf("threshold = %v, should be < max", threshold)
		}
	})

	t.Run("all equal threshold", func(t *testing.T) {
		sorted := []float64{3, 3, 3, 3}
		threshold := sparsemaxFindThreshold(sorted)
		expected := 3.0 - 1.0/4.0
		if !approxEqual(threshold, expected) {
			t.Errorf("threshold = %v, want %v", threshold, expected)
		}
	})

	t.Run("single element", func(t *testing.T) {
		sorted := []float64{7}
		threshold := sparsemaxFindThreshold(sorted)
		expected := 6.0 // (7 - 1) / 1
		if !approxEqual(threshold, expected) {
			t.Errorf("threshold = %v, want %v", threshold, expected)
		}
	})
}

func TestSparsemaxApply(t *testing.T) {
	t.Run("above threshold preserved", func(t *testing.T) {
		src := []float64{3, 2, 1}
		dst := make([]float64, 3)
		sparsemaxApply(src, 1.5, dst)
		if !approxEqual(dst[0], 1.5) {
			t.Errorf("dst[0] = %v, want 1.5", dst[0])
		}
		if !approxEqual(dst[1], 0.5) {
			t.Errorf("dst[1] = %v, want 0.5", dst[1])
		}
	})

	t.Run("below threshold zeroed", func(t *testing.T) {
		src := []float64{1, 2, 3}
		dst := make([]float64, 3)
		sparsemaxApply(src, 2.5, dst)
		if dst[0] != 0 {
			t.Errorf("dst[0] = %v, want 0", dst[0])
		}
		if dst[1] != 0 {
			t.Errorf("dst[1] = %v, want 0", dst[1])
		}
		if !approxEqual(dst[2], 0.5) {
			t.Errorf("dst[2] = %v, want 0.5", dst[2])
		}
	})
}
