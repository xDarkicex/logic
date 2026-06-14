package fuzzy

import (
	"errors"
	"math"
	"math/rand"

	"github.com/xDarkicex/memory"
)

var ErrInsufficientData = errors.New("insufficient data for fuzzy clustering (N < 100)")

// DistanceFunc defines the metric for cluster distance.
type DistanceFunc func(a, b []float64) float64

// EuclideanDistance computes L2 norm distance between a and b.
func EuclideanDistance(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// CosineDistance computes cosine distance (1 - cosine similarity).
// Guaranteed to be >= 0.
func CosineDistance(a, b []float64) float64 {
	dot := 0.0
	normA := 0.0
	normB := 0.0
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 1.0 // Maximum distance if zero vector
	}
	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	// Clamp simulation to [-1, 1] for safety
	if sim > 1.0 {
		sim = 1.0
	} else if sim < -1.0 {
		sim = -1.0
	}
	return 1.0 - sim
}

// FCMConfig configures the clustering run.
type FCMConfig struct {
	Fuzziness    float64
	Epsilon      float64
	NumCentroids int
	Seed         int64
	Distance     DistanceFunc
	Pool         *memory.Pool
	Arena        *memory.Arena
}

// Cluster runs FCM from scratch.
// It returns the centroids and a flat N*C weight matrix from Pool.
func Cluster(vals [][]float64, cfg FCMConfig) ([][]float64, []float64, error) {
	if len(vals) < 100 {
		return nil, nil, ErrInsufficientData
	}

	centroids := memory.MustArenaSlice[[]float64](cfg.Arena, cfg.NumCentroids)[:cfg.NumCentroids]
	initCentroids(vals, cfg.NumCentroids, centroids, cfg.Seed)
	
	weights := ClusterGivenCentroids(vals, centroids, cfg)
	return centroids, weights, nil
}

// ClusterGivenCentroids iterates the FCM algorithm given initial centroids.
// Returns a flat weights slice (size C*N) from cfg.Pool.
func ClusterGivenCentroids(vals [][]float64, centroids [][]float64, cfg FCMConfig) []float64 {
	N := len(vals)
	C := len(centroids)
	
	// weights matrix flattened: weights[c*N + i]
	weights := memory.MustPoolSlice[float64](cfg.Pool, C*N)[:C*N]
	
	for {
		delta := evaluateWeights(vals, cfg.Fuzziness, centroids, weights, cfg)
		if delta <= cfg.Epsilon {
			break
		}
		recenter(vals, cfg.Fuzziness, centroids, weights, cfg)
	}

	return weights
}

// initCentroids initializes centroids reproducibly using the given seed.
// CC <= 8.
func initCentroids(vals [][]float64, numCentroids int, centroids [][]float64, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	N := len(vals)

	if N < 10000 || float64(numCentroids)/float64(N) > 0.2 {
		// Fisher-Yates shuffle
		indices := rng.Perm(N)
		for i := 0; i < numCentroids; i++ {
			centroids[i] = vals[indices[i]]
		}
	} else {
		// Rejection sampling
		seen := make(map[int]bool)
		for i := 0; i < numCentroids; i++ {
			idx := rng.Intn(N)
			for seen[idx] {
				idx = rng.Intn(N)
			}
			seen[idx] = true
			centroids[i] = vals[idx]
		}
	}
}

// evaluateWeights updates the weights matrix and returns the delta (RMS).
// CC <= 8.
func evaluateWeights(vals [][]float64, fuzziness float64, centroids [][]float64, weights []float64, cfg FCMConfig) float64 {
	N := len(vals)
	C := len(centroids)
	squareSum := 0.0

	scratch := memory.MustPoolSlice[float64](cfg.Pool, C)[:C]

	for i := 0; i < N; i++ {
		PredictMembershipInPlace(vals[i], centroids, fuzziness, cfg.Distance, scratch)
		for c := 0; c < C; c++ {
			idx := c*N + i
			diff := weights[idx] - scratch[c]
			squareSum += diff * diff
			weights[idx] = scratch[c]
		}
	}
	return math.Sqrt(squareSum)
}

// PredictMembership evaluates the membership of a single point against centroids.
func PredictMembership(val []float64, centroids [][]float64, fuzziness float64, cfg FCMConfig) []float64 {
	out := memory.MustPoolSlice[float64](cfg.Pool, len(centroids))[:len(centroids)]
	PredictMembershipInPlace(val, centroids, fuzziness, cfg.Distance, out)
	return out
}

// PredictMembershipInPlace performs the membership computation in a pre-allocated slice.
func PredictMembershipInPlace(val []float64, centroids [][]float64, fuzziness float64, dist DistanceFunc, out []float64) {
	for c, centroid := range centroids {
		denominator := 0.0
		for _, c2 := range centroids {
			d1 := dist(val, centroid)
			d2 := dist(val, c2)
			
			// Handle identical points / division by zero
			if d1 == 0 {
				denominator += 0.0 // the c2 loop will yield exactly one case where d2=0, rest d1/d2=0
			} else if d2 == 0 {
				denominator = math.Inf(1) // Avoid NaN, forces out[c]=0
				break
			} else {
				ratio := d1 / d2
				denominator += math.Pow(ratio, 2.0/(fuzziness-1.0))
			}
		}
		
		if math.IsNaN(denominator) || denominator == 0 {
			// If d1=0 and denominator loop completed with d1=0, we are exactly on the centroid.
			out[c] = 1.0
		} else if math.IsInf(denominator, 1) {
			out[c] = 0.0
		} else {
			out[c] = 1.0 / denominator
		}
	}
	
	// Normalize (if val is exactly on a centroid, multiple could have 1.0 theoretically, but handled by d1==0 check above)
	// We make sure they sum to 1.0 if we hit exactly one centroid
	sum := 0.0
	for _, w := range out {
		sum += w
	}
	if sum > 0 {
		for c := range out {
			out[c] /= sum
		}
	}
}

// recenter computes the new centroids.
// CC <= 8.
func recenter(vals [][]float64, fuzziness float64, centroids [][]float64, weights []float64, cfg FCMConfig) {
	N := len(vals)
	C := len(centroids)
	D := len(vals[0])

	for c := 0; c < C; c++ {
		normalization := 0.0
		newC := memory.MustArenaSlice[float64](cfg.Arena, D)[:D] // Temp array zeroed
		
		for i := 0; i < N; i++ {
			w := weights[c*N+i]
			fuzziedWeight := math.Pow(w, fuzziness)
			normalization += fuzziedWeight

			val := vals[i]
			for d := 0; d < D; d++ {
				newC[d] += val[d] * fuzziedWeight
			}
		}

		if normalization > 0 {
			for d := 0; d < D; d++ {
				newC[d] /= normalization
			}
		}
		centroids[c] = newC
	}
}

// UpdateCentroids applies SPFCM logic to update existing centroids incrementally.
func UpdateCentroids(newData [][]float64, oldCentroids [][]float64, flatOldWeights []float64, oldN int, cfg FCMConfig) ([][]float64, []float64) {
	C := len(oldCentroids)
	
	// Compute historical rho for each centroid
	rhos := memory.MustPoolSlice[float64](cfg.Pool, C)[:C]
	
	for c := 0; c < C; c++ {
		sumU := 0.0
		for i := 0; i < oldN; i++ {
			sumU += flatOldWeights[c*oldN+i]
		}
		rhos[c] = sumU
	}

	// Cluster the new data with old centroids as starting point
	newWeights := ClusterGivenCentroids(newData, oldCentroids, cfg)
	newN := len(newData)
	D := len(newData[0])

	// Final recentering incorporating historical virtual weights
	for c := 0; c < C; c++ {
		varC := memory.MustArenaSlice[float64](cfg.Arena, D)[:D]
		norm := 0.0
		
		// Add historical weight acting at the old centroid location
		histWeight := math.Pow(rhos[c], cfg.Fuzziness) // Approximate formulation
		norm += histWeight
		for d := 0; d < D; d++ {
			varC[d] += oldCentroids[c][d] * histWeight
		}

		// Add new data
		for i := 0; i < newN; i++ {
			w := newWeights[c*newN+i]
			fw := math.Pow(w, cfg.Fuzziness)
			norm += fw
			for d := 0; d < D; d++ {
				varC[d] += newData[i][d] * fw
			}
		}

		if norm > 0 {
			for d := 0; d < D; d++ {
				varC[d] /= norm
			}
		}
		oldCentroids[c] = varC
	}

	return oldCentroids, newWeights
}

// FPC computes Fuzzy Partition Coefficient (validity index).
// Maximizing FPC suggests better clustering.
func FPC(flatWeights []float64, C, N int) float64 {
	sum := 0.0
	for _, w := range flatWeights {
		sum += w * w
	}
	return sum / float64(N)
}

// XieBeni computes Xie-Beni validity index.
// Minimizing XB suggests better clustering.
func XieBeni(vals [][]float64, centroids [][]float64, flatWeights []float64, cfg FCMConfig) float64 {
	N := len(vals)
	C := len(centroids)
	
	if C < 2 {
		return math.Inf(1)
	}

	minCentroidDist := math.Inf(1)
	for i := 0; i < C; i++ {
		for j := i + 1; j < C; j++ {
			d := cfg.Distance(centroids[i], centroids[j])
			d2 := d * d
			if d2 < minCentroidDist {
				minCentroidDist = d2
			}
		}
	}

	if minCentroidDist == 0 {
		return math.Inf(1)
	}

	numerator := 0.0
	for c := 0; c < C; c++ {
		for i := 0; i < N; i++ {
			w := flatWeights[c*N+i]
			um := math.Pow(w, cfg.Fuzziness)
			d := cfg.Distance(vals[i], centroids[c])
			numerator += um * (d * d)
		}
	}

	return numerator / (float64(N) * minCentroidDist)
}

// OptimizeClusterCount loops over candidate counts to find the optimal C.
func OptimizeClusterCount(vals [][]float64, maxC int, cfg FCMConfig) int {
	N := len(vals)
	if N < 100 {
		return 0
	}

	limit := int(math.Sqrt(float64(N)))
	if limit > maxC {
		limit = maxC
	}
	if limit < 2 {
		return 2
	}

	bestC := 2
	minXB := math.Inf(1)

	for c := 2; c <= limit; c++ {
		cCfg := cfg
		cCfg.NumCentroids = c
		centroids, weights, err := Cluster(vals, cCfg)
		if err != nil {
			continue
		}

		xb := XieBeni(vals, centroids, weights, cCfg)
		if xb < minXB {
			minXB = xb
			bestC = c
		}
		
	}

	return bestC
}
