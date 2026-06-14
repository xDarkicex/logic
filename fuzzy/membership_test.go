package fuzzy

import (
	"math"
	"testing"
)

func assertClose(t *testing.T, expected, actual TruthValue, msg string) {
	t.Helper()
	if !expected.Equals(actual) {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

func TestTriangular(t *testing.T) {
	mf := Triangular(10, 20, 30)

	assertClose(t, 0.0, mf(5), "below a")
	assertClose(t, 0.0, mf(10), "at a")
	assertClose(t, 0.5, mf(15), "midpoint rise")
	assertClose(t, 1.0, mf(20), "at b")
	assertClose(t, 0.5, mf(25), "midpoint fall")
	assertClose(t, 0.0, mf(30), "at c")
	assertClose(t, 0.0, mf(35), "above c")
}

func TestTrapezoidal(t *testing.T) {
	mf := Trapezoidal(10, 20, 30, 40)

	assertClose(t, 0.0, mf(5), "below a")
	assertClose(t, 0.0, mf(10), "at a")
	assertClose(t, 0.5, mf(15), "midpoint rise")
	assertClose(t, 1.0, mf(20), "at b")
	assertClose(t, 1.0, mf(25), "between b and c")
	assertClose(t, 1.0, mf(30), "at c")
	assertClose(t, 0.5, mf(35), "midpoint fall")
	assertClose(t, 0.0, mf(40), "at d")
	assertClose(t, 0.0, mf(45), "above d")
}

func TestGaussian(t *testing.T) {
	mf := Gaussian(50, 10)

	assertClose(t, 1.0, mf(50), "at mean")
	assertClose(t, TruthValue(math.Exp(-0.5)), mf(60), "1 stddev right")
	assertClose(t, TruthValue(math.Exp(-0.5)), mf(40), "1 stddev left")

	// Zero stddev edge case
	zeroStd := Gaussian(50, 0)
	assertClose(t, 1.0, zeroStd(50), "zero stddev at mean")
	assertClose(t, 0.0, zeroStd(51), "zero stddev off mean")
}

func TestBell(t *testing.T) {
	mf := Bell(20, 2, 50)

	assertClose(t, 1.0, mf(50), "at center")
	assertClose(t, 0.5, mf(70), "at c + a")
	assertClose(t, 0.5, mf(30), "at c - a")

	// a == 0 edge case
	zeroA := Bell(0, 2, 50)
	assertClose(t, 1.0, zeroA(50), "zero a at center")
	assertClose(t, 0.0, zeroA(51), "zero a off center")
}

func TestSigmoid(t *testing.T) {
	mf := Sigmoid(2, 50)

	assertClose(t, 0.5, mf(50), "at center")
	assertClose(t, TruthValue(1.0/(1.0+math.Exp(-20))), mf(60), "positive")
	assertClose(t, TruthValue(1.0/(1.0+math.Exp(20))), mf(40), "negative")
}

func TestSingleton(t *testing.T) {
	mf := Singleton(42)

	assertClose(t, 1.0, mf(42), "at value")
	assertClose(t, 0.0, mf(41), "below value")
	assertClose(t, 0.0, mf(43), "above value")
}

func TestLaplace(t *testing.T) {
	mf := Laplace(50, 10)

	assertClose(t, 1.0, mf(50), "at loc")
	assertClose(t, TruthValue(math.Exp(-1)), mf(60), "1 scale right")
	assertClose(t, TruthValue(math.Exp(-1)), mf(40), "1 scale left")

	// Zero scale edge case
	zeroScale := Laplace(50, 0)
	assertClose(t, 1.0, zeroScale(50), "zero scale at loc")
	assertClose(t, 0.0, zeroScale(51), "zero scale off loc")
}
