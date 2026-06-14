package fuzzy

import (
	"math"
	"testing"

	"github.com/xDarkicex/memory"
)

func newTestPool(t *testing.T) *memory.Pool {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	return pool
}

func TestSoftplus(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"very negative", -1000, 0},
		{"threshold negative", -708, 0},
		{"moderate negative", -1, math.Log1p(math.Exp(-1))},
		{"zero", 0, math.Log1p(1)},
		{"moderate positive", 2, math.Log1p(math.Exp(2))},
		{"threshold positive", 17, 17},
		{"very positive", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Softplus(tt.input)
			if !approxEqual(got, tt.expected) {
				t.Errorf("Softplus(%v) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSoftmax(t *testing.T) {
	pool := newTestPool(t)

	t.Run("normal case with dst provided", func(t *testing.T) {
		src := []float64{1, 2, 3}
		dst := memory.MustPoolSlice[float64](pool, 3)
		dst = dst[:3]
		Softmax(src, dst, pool)

		var sum float64
		for _, v := range dst {
			sum += v
		}
		if !approxEqual(sum, 1.0) {
			t.Errorf("softmax sum = %v, want 1.0", sum)
		}
		if dst[0] >= dst[1] || dst[1] >= dst[2] {
			t.Errorf("softmax not monotonic: %v", dst)
		}
	})

	t.Run("nil dst allocates", func(t *testing.T) {
		src := []float64{1, 2, 3}
		dst := Softmax(src, nil, pool)
		if len(dst) != 3 {
			t.Errorf("len = %d, want 3", len(dst))
		}
		var sum float64
		for _, v := range dst {
			sum += v
		}
		if !approxEqual(sum, 1.0) {
			t.Errorf("sum = %v, want 1.0", sum)
		}
	})

	t.Run("single element", func(t *testing.T) {
		src := []float64{5}
		dst := Softmax(src, nil, pool)
		if !approxEqual(dst[0], 1.0) {
			t.Errorf("single element softmax = %v, want 1.0", dst[0])
		}
	})

	t.Run("all equal values", func(t *testing.T) {
		src := []float64{2, 2, 2, 2}
		dst := Softmax(src, nil, pool)
		expected := 1.0 / 4.0
		for i, v := range dst {
			if !approxEqual(v, expected) {
				t.Errorf("equal values softmax[%d] = %v, want %v", i, v, expected)
			}
		}
	})

	t.Run("numerical stability with large values", func(t *testing.T) {
		src := []float64{1000, 1000, 1000}
		dst := Softmax(src, nil, pool)
		for _, v := range dst {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Errorf("softmax produced NaN/Inf for large values: %v", dst)
			}
		}
	})
}

func TestLogSoftmax(t *testing.T) {
	pool := newTestPool(t)

	t.Run("normal case", func(t *testing.T) {
		src := []float64{1, 2, 3}
		dst := LogSoftmax(src, nil, pool)

		softmaxDst := Softmax(src, nil, pool)
		for i, lv := range dst {
			if !approxEqual(math.Exp(lv), softmaxDst[i]) {
				t.Errorf("exp(logsoftmax[%d]) = %v, want %v", i, math.Exp(lv), softmaxDst[i])
			}
		}
	})

	t.Run("nil dst allocates", func(t *testing.T) {
		src := []float64{1, 2}
		dst := LogSoftmax(src, nil, pool)
		if len(dst) != 2 {
			t.Errorf("len = %d, want 2", len(dst))
		}
	})

	t.Run("single element", func(t *testing.T) {
		src := []float64{42}
		dst := LogSoftmax(src, nil, pool)
		if !approxEqual(dst[0], 0.0) {
			t.Errorf("single element logsoftmax = %v, want 0", dst[0])
		}
	})

	t.Run("numerical stability", func(t *testing.T) {
		src := []float64{1000, 1000}
		dst := LogSoftmax(src, nil, pool)
		for _, v := range dst {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Errorf("logsoftmax produced NaN/Inf for large values: %v", dst)
			}
		}
	})
}

func approxEqual(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-9
}
