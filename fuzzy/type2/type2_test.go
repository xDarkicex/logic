package type2

import (
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
	if diff > 1e-9 {
		t.Errorf("%s: expected %v, got %v", msg, expected, actual)
	}
}

func TestType2Types(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	uFs := fuzzy.NewFuzzySet(1, []float64{0, 10, 20}, pool)
	uFs.Members[0] = 0.0
	uFs.Members[1] = 1.0
	uFs.Members[2] = 0.0

	lFs := fuzzy.NewFuzzySet(1, []float64{0, 10, 20}, pool)
	lFs.Members[0] = 0.0
	lFs.Members[1] = 0.5
	lFs.Members[2] = 0.0

	// Valid creation
	t2, err := NewIntervalType2Set(1, uFs, lFs)
	if err != nil {
		t.Fatalf("Unexpected error creating Type-2 set: %v", err)
	}

	// Membership evaluation
	fou := t2.Membership(10)
	if fou.Upper != 1.0 || fou.Lower != 0.5 {
		t.Errorf("Expected FOU [0.5, 1.0], got [%v, %v]", fou.Lower, fou.Upper)
	}

	// Invalid universe length
	badFs := fuzzy.NewFuzzySet(1, []float64{0, 10}, pool)
	_, err = NewIntervalType2Set(1, uFs, badFs)
	if err == nil {
		t.Error("Expected error for mismatched universe lengths")
	}

	// Invalid bounds (Lower > Upper)
	invLFs := fuzzy.NewFuzzySet(1, []float64{0, 10, 20}, pool)
	invLFs.Members[1] = 1.5
	_, err = NewIntervalType2Set(1, uFs, invLFs)
	if err == nil {
		t.Error("Expected error for Lower > Upper")
	}

	// Linguistic Var
	lv := NewType2LinguisticVar(1)
	lv.AddTerm(10, t2)
	term := lv.GetTerm(10)
	if term == nil || term.ID != 1 {
		t.Error("Failed to retrieve term from LinguisticVar")
	}
}

func TestType2Inference(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	engine := NewType2Engine()

	// Variable 1
	v1 := NewType2LinguisticVar(1)
	uFs1 := fuzzy.NewFuzzySet(10, []float64{0, 10, 20}, pool)
	lFs1 := fuzzy.NewFuzzySet(10, []float64{0, 10, 20}, pool)
	uFs1.Members[1] = 0.8
	lFs1.Members[1] = 0.4
	t2_1, _ := NewIntervalType2Set(10, uFs1, lFs1)
	v1.AddTerm(10, t2_1)
	engine.AddVariable(v1)

	// Variable 2 (Output)
	v2 := NewType2LinguisticVar(2)
	uFs2 := fuzzy.NewFuzzySet(20, []float64{100, 200, 300}, pool)
	lFs2 := fuzzy.NewFuzzySet(20, []float64{100, 200, 300}, pool)
	uFs2.Members[1] = 1.0
	lFs2.Members[1] = 0.5
	t2_2, _ := NewIntervalType2Set(20, uFs2, lFs2)
	v2.AddTerm(20, t2_2)
	engine.AddVariable(v2)

	// Rule 1: IF V1 IS T10 THEN V2 IS T20
	r1 := fuzzy.NewFuzzyRule(1.0)
	r1.AddAntecedent(1, 10, false)
	r1.SetConsequent(2, 20)
	engine.AddRule(*r1)

	// Rule 2: IF V1 IS T10 THEN V2 IS T20 (to trigger aggregateSets)
	r2 := fuzzy.NewFuzzyRule(1.0)
	r2.AddAntecedent(1, 10, false)
	r2.SetConsequent(2, 20)
	engine.AddRule(*r2)

	// Evaluate
	inputs := map[fuzzy.VarID]float64{1: 10} // FOU = [0.4, 0.8]
	outSet, err := engine.Evaluate(inputs, pool)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	// Check output footprint
	// For x=200, out = min(FOU, T20) -> Upper = min(0.8, 1.0) = 0.8. Lower = min(0.4, 0.5) = 0.4
	if outSet.Upper.Members[1] != 0.8 || outSet.Lower.Members[1] != 0.4 {
		t.Errorf("Expected FOU [0.4, 0.8], got [%v, %v]", outSet.Lower.Members[1], outSet.Upper.Members[1])
	}

	// Test NieTanDefuzzify
	// x=[100, 200, 300], M = [ (0+0)/2, (0.8+0.4)/2, (0+0)/2 ] = [0, 0.6, 0]
	// Centroid = (200 * 0.6) / 0.6 = 200
	defuzz := NieTanDefuzzify(outSet)
	assertFloatClose(t, 200.0, defuzz, "NieTanDefuzzify")
	
	// Test Defuzzify zero area
	emptyFsU := fuzzy.NewFuzzySet(0, []float64{100}, pool)
	emptyFsL := fuzzy.NewFuzzySet(0, []float64{100}, pool)
	emptyT2, _ := NewIntervalType2Set(0, emptyFsU, emptyFsL)
	assertFloatClose(t, 0.0, NieTanDefuzzify(emptyT2), "NieTan zero")
}

func TestType2InferenceErrors(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, _ := memory.NewPool(cfg)
	defer pool.Free()

	engine := NewType2Engine()

	// Missing antecedent variable
	r := fuzzy.NewFuzzyRule(1.0)
	r.AddAntecedent(1, 10, false)
	engine.AddRule(*r)
	_, err := engine.Evaluate(map[fuzzy.VarID]float64{1: 10}, pool)
	if err == nil {
		t.Error("Expected error for missing variable")
	}

	// Missing term
	v := NewType2LinguisticVar(1)
	engine.AddVariable(v)
	_, err = engine.Evaluate(map[fuzzy.VarID]float64{1: 10}, pool)
	if err == nil {
		t.Error("Expected error for missing term")
	}

	// Missing input
	uFs := fuzzy.NewFuzzySet(10, []float64{0}, pool)
	lFs := fuzzy.NewFuzzySet(10, []float64{0}, pool)
	t2, _ := NewIntervalType2Set(10, uFs, lFs)
	v.AddTerm(10, t2)
	_, err = engine.Evaluate(map[fuzzy.VarID]float64{}, pool)
	if err == nil {
		t.Error("Expected error for missing input")
	}

	// Negated antecedent
	inputs := map[fuzzy.VarID]float64{1: 0}
	rNeg := fuzzy.NewFuzzyRule(1.0)
	rNeg.AddAntecedent(1, 10, true) // negated
	engine.rules = []fuzzy.FuzzyRule{*rNeg}
	
	// Output missing
	rNeg.SetConsequent(2, 20)
	_, err = engine.Evaluate(inputs, pool)
	if err == nil {
		t.Error("Expected error for missing output var")
	}

	// Output term missing
	v2 := NewType2LinguisticVar(2)
	engine.AddVariable(v2)
	_, err = engine.Evaluate(inputs, pool)
	if err == nil {
		t.Error("Expected error for missing output term")
	}

	// No rules fired
	engine.rules = nil
	// no antecedents -> always fires with weight 1.0
	// But let's test a rule that fires with 0 strength
	r4 := fuzzy.NewFuzzyRule(1.0)
	r4.AddAntecedent(1, 10, false)
	engine.AddRule(*r4)
	
	v2.AddTerm(20, t2) // add dummy term
	_, err = engine.Evaluate(inputs, pool) // term 10 gives 0
	if err == nil {
		t.Error("Expected error for no rules fired")
	}

	// Test unconditional rule
	engine.rules = nil
	rUnc := fuzzy.NewFuzzyRule(1.0)
	rUnc.SetConsequent(2, 20)
	engine.AddRule(*rUnc)
	_, err = engine.Evaluate(inputs, pool)
	if err != nil {
		t.Errorf("Expected no error for unconditional rule, got: %v", err)
	}

	// Test multiple conditions
	engine.rules = nil
	rMulti := fuzzy.NewFuzzyRule(1.0)
	rMulti.AddAntecedent(1, 10, false)
	rMulti.AddAntecedent(1, 10, false)
	rMulti.SetConsequent(2, 20)
	engine.AddRule(*rMulti)

	// Set membership to non-zero so it fires
	uFs.Members[0] = 0.5
	lFs.Members[0] = 0.2
	_, err = engine.Evaluate(inputs, pool)
	if err != nil {
		t.Errorf("Unexpected error with multi-cond: %v", err)
	}
}
