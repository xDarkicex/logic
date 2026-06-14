package fuzzy

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func TestMamdaniEngine(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	engine := NewMamdaniEngine(pool)

	// Custom operator
	engine.SetTNorm(ProductTNorm)
	engine.SetTConorm(ProbabilisticTConorm)
	engine.SetImplication(GodelImplication)

	vTemp := NewLinguisticVar(1)
	u := []float64{0, 50, 100}
	fsCold := NewFuzzySet(10, u, pool)
	fsCold.Members[0] = 1.0
	fsCold.Members[1] = 0.5
	fsCold.Members[2] = 0.0
	vTemp.AddTerm(10, fsCold)
	engine.AddVariable(vTemp)

	vFan := NewLinguisticVar(2)
	fsSlow := NewFuzzySet(20, u, pool)
	fsSlow.Members[0] = 1.0
	fsSlow.Members[1] = 0.5
	fsSlow.Members[2] = 0.0
	vFan.AddTerm(20, fsSlow)
	engine.AddVariable(vFan)

	// Rule: If Temp is Cold (var 1, term 10) then Fan is Slow (var 2, term 20)
	rule := NewFuzzyRule(1.0)
	rule.AddAntecedent(1, 10, false)
	rule.SetConsequent(2, 20)
	engine.AddRule(*rule)

	// Add a second rule to trigger aggregateOutputs
	rule2 := NewFuzzyRule(0.8)
	rule2.AddAntecedent(1, 10, false)
	rule2.SetConsequent(2, 20)
	engine.AddRule(*rule2)

	// Test Evaluate
	inputs := map[VarID]float64{1: 50} // Temp=50 -> fsCold membership=0.5
	res, err := engine.Evaluate(inputs, pool)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Output set: Godel implication with a=0.5, b=fsSlow
	// fsSlow = [1.0, 0.5, 0.0]
	// Godel(0.5, 1.0) = 1.0
	// Godel(0.5, 0.5) = 1.0
	// Godel(0.5, 0.0) = 0.0
	assertClose(t, 1.0, res.Members[0], "res 0")
	assertClose(t, 1.0, res.Members[1], "res 1")
	assertClose(t, 0.0, res.Members[2], "res 2")

	// Test missing input
	_, err = engine.Evaluate(map[VarID]float64{99: 50}, pool)
	if err == nil {
		t.Error("Expected error for missing input")
	}

	// Test no rules fired
	inputs2 := map[VarID]float64{1: 100} // fsCold = 0.0 -> rule doesn't fire
	_, err = engine.Evaluate(inputs2, pool)
	if err == nil {
		t.Error("Expected error for no rules fired")
	}
}

func TestMamdaniEngine_Errors(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	engine := NewMamdaniEngine(pool)

	// Missing variable
	rule := NewFuzzyRule(1.0)
	rule.AddAntecedent(99, 10, false)
	engine.AddRule(*rule)
	_, err = engine.Evaluate(map[VarID]float64{99: 0}, pool)
	if err == nil {
		t.Error("Expected error for missing variable")
	}

	// Missing term
	engine2 := NewMamdaniEngine(pool)
	v := NewLinguisticVar(1)
	engine2.AddVariable(v)
	r2 := NewFuzzyRule(1.0)
	r2.AddAntecedent(1, 10, false) // term 10 does not exist
	engine2.AddRule(*r2)
	_, err = engine2.Evaluate(map[VarID]float64{1: 0}, pool)
	if err == nil {
		t.Error("Expected error for missing term")
	}

	// Missing antecedent term evaluation
	v3 := NewLinguisticVar(2)
	fs3 := NewFuzzySet(10, []float64{0}, pool) // term 10 exists
	v3.AddTerm(10, fs3)
	engine2.AddVariable(v3)
	r3 := NewFuzzyRule(1.0)
	r3.AddAntecedent(2, 99, false) // term 99 does not exist
	engine2.AddRule(*r3)
	_, err = engine2.Evaluate(map[VarID]float64{2: 0}, pool)
	if err == nil {
		t.Error("Expected error for missing antecedent term")
	}
}

func TestMamdaniEngine_ConsequentErrors(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	engine := NewMamdaniEngine(pool)
	v := NewLinguisticVar(1)
	fs := NewFuzzySet(10, []float64{0}, pool)
	fs.Members[0] = 1.0
	v.AddTerm(10, fs)
	engine.AddVariable(v)

	r := NewFuzzyRule(1.0)
	r.AddAntecedent(1, 10, false)
	r.SetConsequent(99, 20) // Missing consequent var
	engine.AddRule(*r)

	_, err = engine.Evaluate(map[VarID]float64{1: 0}, pool)
	if err == nil {
		t.Error("Expected error for missing consequent var")
	}

	v2 := NewLinguisticVar(2)
	engine.AddVariable(v2)
	r.SetConsequent(2, 99) // Missing consequent term
	_, err = engine.Evaluate(map[VarID]float64{1: 0}, pool)
	if err == nil {
		t.Error("Expected error for missing consequent term")
	}
}

func TestMamdaniEngine_UnconditionalRule(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	engine := NewMamdaniEngine(pool)
	outVar := NewLinguisticVar(2)
	outSet := NewFuzzySet(20, []float64{0, 10}, pool)
	outSet.Members[0] = 0.5
	outSet.Members[1] = 0.5
	outVar.AddTerm(20, outSet)
	engine.AddVariable(outVar)

	r := NewFuzzyRule(1.0)
	r.SetConsequent(2, 20) // No antecedents
	engine.AddRule(*r)

	res, err := engine.Evaluate(map[VarID]float64{}, pool)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res.Members[0] != 0.5 {
		t.Errorf("Expected 0.5, got %v", res.Members[0])
	}
}

func TestMamdaniEngine_NegatedAntecedent(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	engine := NewMamdaniEngine(pool)
	v1 := NewLinguisticVar(1)
	fs := NewFuzzySet(10, []float64{0}, pool)
	fs.Members[0] = 0.2 // Membership = 0.2
	v1.AddTerm(10, fs)
	engine.AddVariable(v1)

	v2 := NewLinguisticVar(2)
	fsOut := NewFuzzySet(20, []float64{0}, pool)
	fsOut.Members[0] = 1.0
	v2.AddTerm(20, fsOut)
	engine.AddVariable(v2)

	r := NewFuzzyRule(1.0)
	r.AddAntecedent(1, 10, true) // Negated! Should be 0.8
	r.SetConsequent(2, 20)
	engine.AddRule(*r)

	res, err := engine.Evaluate(map[VarID]float64{1: 0}, pool)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	// Min(0.8, 1.0) = 0.8
	if res.Members[0] != 0.8 {
		t.Errorf("Expected 0.8, got %v", res.Members[0])
	}
}

func TestTSKEngine(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	engine := NewTSKEngine()
	engine.SetTNorm(ProductTNorm)

	vTemp := NewLinguisticVar(1)
	fsHot := NewFuzzySet(10, []float64{0, 50, 100}, pool)
	fsHot.Members[0] = 0.0
	fsHot.Members[1] = 0.5
	fsHot.Members[2] = 1.0
	vTemp.AddTerm(10, fsHot)
	engine.AddVariable(vTemp)

	// Rule 1: If Temp is Hot, Fan = 50 (Constant)
	r1 := TSKRule{
		Antecedents: []FuzzyCondition{{Variable: 1, Term: 10, Negated: false}},
		Consequent:  ConstantTSK(50.0),
		Weight:      1.0,
	}
	engine.AddRule(r1)

	// Rule 2: If Temp is Hot (Negated), Fan = 2*Temp + 10 (Linear)
	r2 := TSKRule{
		Antecedents: []FuzzyCondition{{Variable: 1, Term: 10, Negated: true}},
		Consequent:  LinearTSK{Coeffs: map[VarID]float64{1: 2.0}, Intercept: 10.0},
		Weight:      1.0, // test default 0 handled in eval
	}
	engine.AddRule(r2)
	
	// Test default weight 0 handling
	r3 := TSKRule{
		Antecedents: []FuzzyCondition{{Variable: 1, Term: 10, Negated: false}},
		Consequent:  ConstantTSK(0.0),
		Weight:      0.0,
	}
	engine.AddRule(r3)

	inputs := map[VarID]float64{1: 50} // fsHot=0.5, Negated fsHot=0.5
	out, err := engine.Evaluate(inputs)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	// r1: weight=1.0, strength=0.5 -> w=0.5, z=50.0
	// r2: weight=1.0, strength=0.5 -> w=0.5, z=2*50+10 = 110.0
	// r3: weight=0.0, strength=0.5 -> handled as 0 weight default to 1.0? wait, I passed weight=1.0 but it should have defaulted to 1.0. No, I set it to 0.0, so the engine overrides it to 1.0! 
	// The code: if fr.Weight == 0 { fr.Weight = 1.0 }. So r3: weight=1.0, strength=0.5 -> w=0.5, z=0.0
	// total w = 0.5 + 0.5 + 0.5 = 1.5
	// total num = 0.5*50 + 0.5*110 + 0.5*0 = 25 + 55 + 0 = 80
	// out = 80 / 1.5 = 53.333333333333336
	assertClose(t, TruthValue(53.333333333333336), TruthValue(out), "TSK output")

	// No rules fired
	engine2 := NewTSKEngine()
	engine2.AddVariable(vTemp)
	r4 := TSKRule{
		Antecedents: []FuzzyCondition{{Variable: 1, Term: 10, Negated: false}},
		Consequent:  ConstantTSK(10.0),
		Weight:      1.0,
	}
	engine2.AddRule(r4)
	_, err = engine2.Evaluate(map[VarID]float64{1: 0}) // fsHot=0.0 -> no fire
	if err == nil {
		t.Error("Expected error for no rules fired")
	}

	// Antecedent eval error
	_, err = engine2.Evaluate(map[VarID]float64{99: 0})
	if err == nil {
		t.Error("Expected error for missing input")
	}
}
