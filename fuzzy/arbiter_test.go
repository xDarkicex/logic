package fuzzy

import (
	"fmt"
	"testing"

	"github.com/xDarkicex/memory"
)

// mockStrategy is a simple strategy for testing arbiter.
type mockStrategy struct {
	val float64
	err error
}

func (m *mockStrategy) Evaluate(inputs map[VarID]float64, pool *memory.Pool) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.val, nil
}

func TestArbiterSelect(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	arbiter := NewArbiter()

	// Empty arbiter
	_, _, err = arbiter.Select(nil, pool)
	if err == nil {
		t.Error("Expected error for empty arbiter")
	}

	// Add strategies
	arbiter.AddStrategy("StratA", &mockStrategy{val: 10.0}, 0.5)
	arbiter.AddStrategy("StratB", &mockStrategy{val: 20.0}, 0.8) // Highest score
	arbiter.AddStrategy("StratC", &mockStrategy{err: fmt.Errorf("fail")}, 0.9) // Fails

	val, name, err := arbiter.Select(nil, pool)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if name != "StratB" {
		t.Errorf("Expected StratB, got %s", name)
	}
	if val != 20.0 {
		t.Errorf("Expected 20.0, got %v", val)
	}

	// Test UpdateReliability
	err = arbiter.UpdateReliability("StratA", 0.5) // 0.5 + 0.5 = 1.0 (now highest)
	if err != nil {
		t.Fatalf("Failed to update reliability: %v", err)
	}

	val, name, err = arbiter.Select(nil, pool)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if name != "StratA" {
		t.Errorf("Expected StratA, got %s", name)
	}
	if val != 10.0 {
		t.Errorf("Expected 10.0, got %v", val)
	}

	// Update non-existent strategy
	err = arbiter.UpdateReliability("Unknown", 1.0)
	if err == nil {
		t.Error("Expected error for unknown strategy")
	}

	// All strategies fail
	arbiterAllFail := NewArbiter()
	arbiterAllFail.AddStrategy("Fail1", &mockStrategy{err: fmt.Errorf("fail1")}, 1.0)
	_, _, err = arbiterAllFail.Select(nil, pool)
	if err == nil {
		t.Error("Expected error when all strategies fail")
	}
}

func TestArbiterWrappers(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	// Test MamdaniStrategy
	mamEngine := NewMamdaniEngine(pool)
	vTemp := NewLinguisticVar(1)
	fsHot := NewFuzzySet(10, []float64{0, 50, 100}, pool)
	fsHot.Members[0] = 0.0
	fsHot.Members[1] = 0.5
	fsHot.Members[2] = 1.0
	vTemp.AddTerm(10, fsHot)
	mamEngine.AddVariable(vTemp)
	
	r1 := NewFuzzyRule(1.0)
	r1.AddAntecedent(1, 10, false)
	r1.SetConsequent(1, 10) // identity
	mamEngine.AddRule(*r1)

	mamStrat := &MamdaniStrategy{
		Engine:     mamEngine,
		DefuzzFunc: Centroid,
	}

	inputs := map[VarID]float64{1: 100} // Firing strength = 1.0
	val, err := mamStrat.Evaluate(inputs, pool)
	if err != nil {
		t.Fatalf("Mamdani Evaluate error: %v", err)
	}
	// fsHot Centroid -> (0*0 + 50*0.5 + 100*1) / (0 + 0.5 + 1) = 125 / 1.5 = 83.33333333333333
	assertFloatClose(t, 83.33333333333333, val, "MamdaniStrategy output")

	// Test Mamdani Error
	_, err = mamStrat.Evaluate(map[VarID]float64{99: 100}, pool) // missing input
	if err == nil {
		t.Error("Expected error from MamdaniStrategy")
	}

	// Test TSKStrategy
	tskEngine := NewTSKEngine()
	tskEngine.AddVariable(vTemp)
	rTSK := TSKRule{
		Antecedents: []FuzzyCondition{{Variable: 1, Term: 10, Negated: false}},
		Consequent:  ConstantTSK(42.0),
		Weight:      1.0,
	}
	tskEngine.AddRule(rTSK)

	tskStrat := &TSKStrategy{
		Engine: tskEngine,
	}

	val, err = tskStrat.Evaluate(inputs, pool)
	if err != nil {
		t.Fatalf("TSK Evaluate error: %v", err)
	}
	if val != 42.0 {
		t.Errorf("Expected 42.0, got %v", val)
	}
}
