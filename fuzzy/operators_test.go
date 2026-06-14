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

func TestAdditionalTNorms(t *testing.T) {
	// EinsteinProduct: (a*b)/(2-(a+b-a*b))
	assertClose(t, TruthValue(0.3*0.7/(2.0-(0.3+0.7-0.3*0.7))), EinsteinProduct(0.3, 0.7), "Einstein 0.3 0.7")

	// HamacherProduct: (a*b)/(a+b-a*b)
	assertClose(t, 0.0, HamacherProduct(0.0, 0.0), "Hamacher 0 0")
	assertClose(t, TruthValue(0.3*0.7/(0.3+0.7-0.3*0.7)), HamacherProduct(0.3, 0.7), "Hamacher 0.3 0.7")

	// NilpotentMinimum: min(a,b) if a+b>1 else 0
	assertClose(t, 0.3, NilpotentMinimum(0.3, 0.8), "Nilpotent 0.3 0.8 (sum>1)")
	assertClose(t, 0.0, NilpotentMinimum(0.3, 0.5), "Nilpotent 0.3 0.5 (sum<1)")

	// DrasticProduct: min(a,b) if max(a,b)==1 else 0
	assertClose(t, 0.5, DrasticProduct(1.0, 0.5), "Drastic 1.0 0.5")
	assertClose(t, 0.5, DrasticProduct(0.5, 1.0), "Drastic 0.5 1.0")
	assertClose(t, 0.0, DrasticProduct(0.3, 0.7), "Drastic 0.3 0.7")
}

func TestAdditionalTConorms(t *testing.T) {
	// EinsteinSum: (a+b)/(1+a*b)
	assertClose(t, TruthValue((0.3+0.7)/(1.0+0.3*0.7)), EinsteinSum(0.3, 0.7), "Einstein 0.3 0.7")

	// HamacherSum: (a+b-2ab)/(1-ab)
	assertClose(t, 1.0, HamacherSum(1.0, 1.0), "Hamacher 1 1")
	assertClose(t, TruthValue((0.3+0.7-2.0*0.3*0.7)/(1.0-0.3*0.7)), HamacherSum(0.3, 0.7), "Hamacher 0.3 0.7")

	// NilpotentMaximum: max(a,b) if a+b<1 else 1
	assertClose(t, 0.5, NilpotentMaximum(0.3, 0.5), "Nilpotent 0.3 0.5 (sum<1 => max=0.5)")
	assertClose(t, 1.0, NilpotentMaximum(0.7, 0.8), "Nilpotent 0.7 0.8 (sum>1)")

	// DrasticSum: max(a,b) if min(a,b)==0 else 1
	assertClose(t, 0.5, DrasticSum(0.0, 0.5), "Drastic 0.0 0.5")
	assertClose(t, 0.5, DrasticSum(0.5, 0.0), "Drastic 0.5 0.0")
	assertClose(t, 1.0, DrasticSum(0.3, 0.7), "Drastic 0.3 0.7")

	// NormalizedSum: (a+b)/max(1,a+b)
	assertClose(t, 0.8, NormalizedSum(0.3, 0.5), "Normalized 0.3 0.5 (sum<=1)")
	assertClose(t, 1.0, NormalizedSum(0.7, 0.8), "Normalized 0.7 0.8 (sum>1)")

	// UnboundedSum: a+b
	assertClose(t, 1.0, UnboundedSum(0.3, 0.7), "Unbounded 0.3 0.7")
	assertClose(t, 1.5, UnboundedSum(0.8, 0.7), "Unbounded 0.8 0.7")
}
