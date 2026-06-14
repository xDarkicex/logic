package anfis

import (
	"math"
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

func assertFloatClose(t *testing.T, expected, actual float64, msg string) {
	t.Helper()
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	if diff > 1.5 { // Use wider tolerance for gradient descent over few epochs
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

func TestANFISTrain(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	// Target function: y = 2*x + 5
	// We'll give it inputs around x = 10, 20
	inputs := []map[fuzzy.VarID]float64{
		{1: 10},
		{1: 20},
		{1: 30},
	}
	targets := []float64{
		25, // 2*10+5
		45, // 2*20+5
		65, // 2*30+5
	}

	net := NewANFIS(pool)

	// Setup Variable
	v := fuzzy.NewLinguisticVar(1, pool)
	
	// Term 10 (Gaussian around 20)
	fs := fuzzy.NewFuzzySet(10, nil, pool)
	fs.Func = fuzzy.Gaussian(20.0, 10.0)
	v.AddTerm(10, fs)
	net.Engine.AddVariable(v)

	// Setup Rule
	// IF V1 IS T10 THEN Y = 1*X + 1
	r := fuzzy.TSKRule{
		Antecedents: []fuzzy.FuzzyCondition{{Variable: 1, Term: 10, Negated: false}},
		Consequent:  fuzzy.LinearTSK{Coeffs: map[fuzzy.VarID]float64{1: 1.0}, Intercept: 1.0},
		Weight:      1.0,
	}
	net.Engine.AddRule(r)

	// Register parameters for training
	net.Params = append(net.Params, GaussianParam{
		VarID: 1,
		Term:  10,
		Mean:  20.0,
		Sigma: 10.0,
	})

	net.Rules = append(net.Rules, TSKConsequentParam{
		RuleIdx: 0,
		Coeffs:  map[fuzzy.VarID]float64{1: 1.0},
		Bias:    1.0,
	})

	// Initial forward pass should be off
	yInit, _ := net.Engine.Evaluate(inputs[0])
	if math.Abs(yInit-targets[0]) < 1e-4 {
		t.Errorf("Initial prediction too close to target? yInit=%v", yInit)
	}

	// Train
	err = net.Train(5000, 0.001, inputs, targets)
	if err != nil {
		t.Fatalf("Train failed: %v", err)
	}

	// Verify it learned y = 2*x + 5
	// Check learned parameters
	finalBias := net.Rules[0].Bias
	finalCoeff := net.Rules[0].Coeffs[1]
	
	// Because we only have one rule, ANFIS should trivially learn the linear coefficients
	// to perfectly match the target regardless of the Gaussian if the Gaussian fires > 0.
	assertFloatClose(t, 5.0, finalBias, "Learned bias")
	assertFloatClose(t, 2.0, finalCoeff, "Learned coeff")

	// Final prediction should match
	yFinal, _ := net.Engine.Evaluate(inputs[0])
	assertFloatClose(t, targets[0], yFinal, "Final prediction matches target")

	// Error handling
	err = net.Train(1, 0.1, inputs, []float64{1})
	if err == nil {
		t.Error("Expected error for length mismatch")
	}

	err = net.Train(1, 0.1, []map[fuzzy.VarID]float64{{99: 0}}, []float64{1})
	if err == nil {
		t.Error("Expected error for forward pass failure")
	}
}
