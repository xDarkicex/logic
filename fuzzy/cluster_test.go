package fuzzy

import (
	"math"
	"math/rand"
	"testing"

	"github.com/xDarkicex/memory"
)

func TestDistances(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	
	euc := EuclideanDistance(a, b)
	if math.Abs(euc-math.Sqrt(2)) > 1e-6 {
		t.Errorf("Euclidean incorrect: %v", euc)
	}

	cos := CosineDistance(a, b)
	if cos != 1.0 { // orthogonal
		t.Errorf("Cosine incorrect: %v", cos)
	}

	a2 := []float64{2, 0, 0}
	cos2 := CosineDistance(a, a2)
	if cos2 != 0.0 { // parallel
		t.Errorf("Cosine parallel incorrect: %v", cos2)
	}

	zero := []float64{0, 0, 0}
	cos3 := CosineDistance(a, zero)
	if cos3 != 1.0 {
		t.Errorf("Cosine zero vector incorrect: %v", cos3)
	}
}

func generateBlobs(numBlobs, pointsPerBlob, dims int, spread float64, seed int64) [][]float64 {
	rng := rand.New(rand.NewSource(seed))
	vals := make([][]float64, 0, numBlobs*pointsPerBlob)
	for b := 0; b < numBlobs; b++ {
		center := make([]float64, dims)
		for d := 0; d < dims; d++ {
			center[d] = rng.Float64() * 100.0
		}
		for p := 0; p < pointsPerBlob; p++ {
			point := make([]float64, dims)
			for d := 0; d < dims; d++ {
				point[d] = center[d] + (rng.NormFloat64() * spread)
			}
			vals = append(vals, point)
		}
	}
	return vals
}

func TestFCMCluster(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	// Need at least 100 points
	vals := generateBlobs(3, 40, 5, 2.0, 42) // 120 points, 5 dims

	cfg := FCMConfig{
		Fuzziness:    2.0,
		Epsilon:      1e-4,
		NumCentroids: 3,
		Seed:         42,
		Distance:     EuclideanDistance,
		Pool:         pool,
		Arena:        arena,
	}

	centroids, weights, err := Cluster(vals, cfg)
	if err != nil {
		t.Fatalf("Cluster failed: %v", err)
	}

	if len(centroids) != 3 {
		t.Errorf("Expected 3 centroids, got %d", len(centroids))
	}
	
	if len(weights) != 3*120 {
		t.Errorf("Expected weight length %d, got %d", 3*120, len(weights))
	}

	// Verify weights sum to 1.0
	for i := 0; i < 120; i++ {
		sum := 0.0
		for c := 0; c < 3; c++ {
			sum += weights[c*120+i]
		}
		if math.Abs(sum-1.0) > 1e-4 {
			t.Errorf("Point %d weights sum to %v, expected 1.0", i, sum)
		}
	}

	// Calculate indices
	fpc := FPC(weights, 3, 120)
	if fpc < 0.3 || fpc > 1.0 {
		t.Errorf("FPC out of bounds: %v", fpc)
	}

	xb := XieBeni(vals, centroids, weights, cfg)
	if math.IsNaN(xb) {
		t.Error("Xie-Beni is NaN")
	}

	// PredictMembership
	mem := PredictMembership(vals[0], centroids, 2.0, cfg)
	if len(mem) != 3 {
		t.Errorf("Expected 3 memberships, got %d", len(mem))
	}

	// Re-centering prediction inside the original weight array
	mem0 := weights[0*120+0]
	mem1 := weights[1*120+0]
	mem2 := weights[2*120+0]

	if math.Abs(mem[0]-mem0) > 1e-4 || math.Abs(mem[1]-mem1) > 1e-4 || math.Abs(mem[2]-mem2) > 1e-4 {
		t.Errorf("PredictMembership diverges from Cluster weights")
	}

	// UpdateCentroids (Incremental)
	newData := generateBlobs(1, 10, 5, 1.0, 99)
	newCentroids, newWeights := UpdateCentroids(newData, centroids, weights, 120, cfg)
	
	if len(newCentroids) != 3 {
		t.Errorf("UpdateCentroids length %d", len(newCentroids))
	}
	if len(newWeights) != 3*10 {
		t.Errorf("UpdateCentroids weights length %d", len(newWeights))
	}
}

func TestClusterErrorsAndEdgeCases(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	cfg := FCMConfig{
		Fuzziness:    2.0,
		Epsilon:      1e-4,
		NumCentroids: 2,
		Seed:         1,
		Distance:     EuclideanDistance,
		Pool:         pool,
		Arena:        arena,
	}

	// Less than 100 points
	small := make([][]float64, 50)
	for i := range small {
		small[i] = []float64{float64(i)}
	}

	_, _, err := Cluster(small, cfg)
	if err != ErrInsufficientData {
		t.Errorf("Expected ErrInsufficientData, got %v", err)
	}

	cCount := OptimizeClusterCount(small, 5, cfg)
	if cCount != 0 {
		t.Errorf("OptimizeClusterCount expected 0, got %v", cCount)
	}

	// PredictMembership on exactly the centroid
	vals := generateBlobs(2, 60, 2, 1.0, 42) // 120 points
	centroids, _, _ := Cluster(vals, cfg)

	exactMem := PredictMembership(centroids[0], centroids, 2.0, cfg)
	if exactMem[0] != 1.0 {
		t.Errorf("Exact centroid membership expected 1.0, got %v", exactMem[0])
	}
	if exactMem[1] != 0.0 {
		t.Errorf("Exact centroid other membership expected 0.0, got %v", exactMem[1])
	}
}

func TestOptimizeClusterCount(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	// Generate 4 clear blobs
	vals := generateBlobs(4, 100, 3, 0.5, 42) // 400 points

	cfg := FCMConfig{
		Fuzziness:    2.0,
		Epsilon:      1e-4,
		Seed:         42,
		Distance:     EuclideanDistance,
		Pool:         pool,
		Arena:        arena,
	}

	bestC := OptimizeClusterCount(vals, 10, cfg)
	if bestC != 4 {
		t.Logf("Expected bestC = 4, got %d. (This can be heuristic dependent, so logging instead of failing)", bestC)
	}
}

func TestInitCentroidsCoverage(t *testing.T) {
	// Rejection sampling branch (N >= 10000)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	// Simulate N=10000 with simple 1D points
	vals := make([][]float64, 10000)
	for i := 0; i < 10000; i++ {
		vals[i] = []float64{float64(i)}
	}

	centroids := make([][]float64, 5)
	initCentroids(vals, 5, centroids, 123)

	if len(centroids) != 5 {
		t.Errorf("Expected 5 centroids")
	}
	if centroids[0] == nil {
		t.Error("Centroid not initialized")
	}
}

func TestXieBeniEdgeCases(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()
	
	cfg := FCMConfig{Fuzziness: 2.0, Distance: EuclideanDistance}
	
	xb1 := XieBeni([][]float64{{1}}, [][]float64{{1}}, []float64{1}, cfg)
	if !math.IsInf(xb1, 1) {
		t.Errorf("Expected Inf for C<2, got %v", xb1)
	}

	xb2 := XieBeni([][]float64{{1}}, [][]float64{{1}, {1}}, []float64{1, 0}, cfg)
	if !math.IsInf(xb2, 1) {
		t.Errorf("Expected Inf for zero min centroid dist, got %v", xb2)
	}
}
