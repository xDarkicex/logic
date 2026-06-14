package fuzzy

import (
	"testing"
)

func TestTNorms(t *testing.T) {
	// Min
	assertClose(t, 0.3, MinTNorm(0.3, 0.7), "Min 0.3 0.7")
	assertClose(t, 0.3, MinTNorm(0.7, 0.3), "Min 0.7 0.3")
	assertClose(t, 0.0, MinTNorm(0.0, 0.5), "Min 0.0 0.5")

	// Product
	assertClose(t, 0.21, ProductTNorm(0.3, 0.7), "Product 0.3 0.7")
	assertClose(t, 0.0, ProductTNorm(0.0, 0.7), "Product 0.0 0.7")

	// Lukasiewicz
	assertClose(t, 0.0, LukasiewiczTNorm(0.3, 0.5), "LukasiewiczTNorm 0.3 0.5")
	assertClose(t, 0.2, LukasiewiczTNorm(0.5, 0.7), "LukasiewiczTNorm 0.5 0.7")

	// Min Variadic
	assertClose(t, 1.0, MinTNormVariadic(), "MinVariadic empty")
	assertClose(t, 0.5, MinTNormVariadic(0.5), "MinVariadic single")
	assertClose(t, 0.2, MinTNormVariadic(0.8, 0.5, 0.2, 0.9), "MinVariadic multiple")
}

func TestTConorms(t *testing.T) {
	// Max
	assertClose(t, 0.7, MaxTConorm(0.3, 0.7), "Max 0.3 0.7")
	assertClose(t, 0.7, MaxTConorm(0.7, 0.3), "Max 0.7 0.3")
	assertClose(t, 0.5, MaxTConorm(0.0, 0.5), "Max 0.0 0.5")

	// Probabilistic
	assertClose(t, 0.79, ProbabilisticTConorm(0.3, 0.7), "Probabilistic 0.3 0.7")

	// Lukasiewicz
	assertClose(t, 0.8, LukasiewiczTConorm(0.3, 0.5), "LukasiewiczTConorm 0.3 0.5")
	assertClose(t, 1.0, LukasiewiczTConorm(0.5, 0.7), "LukasiewiczTConorm 0.5 0.7")

	// Max Variadic
	assertClose(t, 0.0, MaxTConormVariadic(), "MaxVariadic empty")
	assertClose(t, 0.5, MaxTConormVariadic(0.5), "MaxVariadic single")
	assertClose(t, 0.9, MaxTConormVariadic(0.8, 0.5, 0.2, 0.9), "MaxVariadic multiple")
}

func TestNegation(t *testing.T) {
	assertClose(t, 0.7, StandardNegation(0.3), "StandardNegation 0.3")
	assertClose(t, 1.0, StandardNegation(0.0), "StandardNegation 0.0")
	assertClose(t, 0.0, StandardNegation(1.0), "StandardNegation 1.0")
}

func TestImplications(t *testing.T) {
	// Godel
	assertClose(t, 1.0, GodelImplication(0.3, 0.7), "Godel 0.3 -> 0.7")
	assertClose(t, 1.0, GodelImplication(0.5, 0.5), "Godel 0.5 -> 0.5")
	assertClose(t, 0.3, GodelImplication(0.7, 0.3), "Godel 0.7 -> 0.3")

	// Goguen
	assertClose(t, 1.0, GoguenImplication(0.0, 0.5), "Goguen 0.0 -> 0.5")
	assertClose(t, 1.0, GoguenImplication(0.3, 0.7), "Goguen 0.3 -> 0.7")
	assertClose(t, TruthValue(0.3/0.7), GoguenImplication(0.7, 0.3), "Goguen 0.7 -> 0.3")

	// Lukasiewicz
	assertClose(t, 1.0, LukasiewiczImplication(0.3, 0.7), "Lukasiewicz 0.3 -> 0.7")
	assertClose(t, 0.6, LukasiewiczImplication(0.7, 0.3), "Lukasiewicz 0.7 -> 0.3")

	// Kleene-Dienes
	assertClose(t, 0.7, KleeneDienesImplication(0.3, 0.5), "KleeneDienes 0.3 -> 0.5")
	assertClose(t, 0.5, KleeneDienesImplication(0.7, 0.5), "KleeneDienes 0.7 -> 0.5")
}
