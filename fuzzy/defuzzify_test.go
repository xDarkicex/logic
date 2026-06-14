package fuzzy

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func assertFloatClose(t *testing.T, expected, actual float64, msg string) {
	t.Helper()
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff > 1e-9 {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

func TestDefuzzifyMethods(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	// Symmetric set around 20
	universe := []float64{0, 10, 20, 30, 40}
	fs := NewFuzzySet(1, universe, pool)
	fs.Members[0] = 0.0
	fs.Members[1] = 0.5
	fs.Members[2] = 1.0
	fs.Members[3] = 0.5
	fs.Members[4] = 0.0

	// Centroid
	assertFloatClose(t, 20.0, Centroid(fs), "Centroid")

	// MeanOfMax
	assertFloatClose(t, 20.0, MeanOfMax(fs), "MeanOfMax single peak")

	// SmallestOfMax / LargestOfMax
	assertFloatClose(t, 20.0, SmallestOfMax(fs), "SOM single peak")
	assertFloatClose(t, 20.0, LargestOfMax(fs), "LOM single peak")

	// Bisector
	// Area = 0 + 0.5 + 1.0 + 0.5 + 0 = 2.0. Target = 1.0.
	// At i=1, area=0.5. At i=2, area=1.5 >= 1.0 -> 20.
	assertFloatClose(t, 20.0, Bisector(fs), "Bisector")

	// Empty set
	emptyFs := NewFuzzySet(2, nil, pool)
	assertFloatClose(t, 0.0, Centroid(emptyFs), "Centroid empty")
	assertFloatClose(t, 0.0, MeanOfMax(emptyFs), "MOM empty")
	assertFloatClose(t, 0.0, SmallestOfMax(emptyFs), "SOM empty")
	assertFloatClose(t, 0.0, LargestOfMax(emptyFs), "LOM empty")
	assertFloatClose(t, 0.0, Bisector(emptyFs), "Bisector empty")

	// Flat top
	fsFlat := NewFuzzySet(3, universe, pool)
	fsFlat.Members[0] = 0.0
	fsFlat.Members[1] = 1.0
	fsFlat.Members[2] = 1.0
	fsFlat.Members[3] = 1.0
	fsFlat.Members[4] = 0.0
	
	assertFloatClose(t, 20.0, MeanOfMax(fsFlat), "MOM flat top")
	assertFloatClose(t, 10.0, SmallestOfMax(fsFlat), "SOM flat top")
	assertFloatClose(t, 30.0, LargestOfMax(fsFlat), "LOM flat top")

	// Bisector fallback
	fsAllZero := NewFuzzySet(4, []float64{10, 20, 30}, pool)
	assertFloatClose(t, 0.0, Bisector(fsAllZero), "Bisector all zero")
}

func TestWeightedAverageDefuzz(t *testing.T) {
	values := []float64{10, 20, 30}
	weights := []float64{0.2, 0.5, 0.3}

	// 10*0.2 + 20*0.5 + 30*0.3 = 2 + 10 + 9 = 21
	// 0.2 + 0.5 + 0.3 = 1.0
	// 21 / 1 = 21
	assertFloatClose(t, 21.0, WeightedAverageDefuzz(values, weights), "WeightedAverage")

	// Zero sum
	assertFloatClose(t, 0.0, WeightedAverageDefuzz(values, []float64{0, 0, 0}), "Zero weights")
}
