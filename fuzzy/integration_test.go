package fuzzy

import (
	"testing"
	"time"

	"github.com/xDarkicex/memory"
)

// Helper to set up tipping problem (Mamdani)
func setupTipping(pool *memory.Pool, arena *memory.Arena) (*MamdaniEngine, *SymbolTable) {
	sym := NewSymbolTable(arena)
	engine := NewMamdaniEngine(pool)

	// Variables
	vService := NewLinguisticVar(sym.Register("Service", arena))
	vFood := NewLinguisticVar(sym.Register("Food", arena))
	vTip := NewLinguisticVar(sym.Register("Tip", arena))

	// Service terms
	fsS_Poor := NewFuzzySet(sym.Register("Poor", arena), nil, pool)
	fsS_Poor.Func = Gaussian(0, 1.5)
	fsS_Good := NewFuzzySet(sym.Register("Good", arena), nil, pool)
	fsS_Good.Func = Gaussian(5, 1.5)
	fsS_Excellent := NewFuzzySet(sym.Register("Excellent", arena), nil, pool)
	fsS_Excellent.Func = Gaussian(10, 1.5)
	
	vService.AddTerm(fsS_Poor.ID, fsS_Poor)
	vService.AddTerm(fsS_Good.ID, fsS_Good)
	vService.AddTerm(fsS_Excellent.ID, fsS_Excellent)

	// Food terms
	fsF_Rancid := NewFuzzySet(sym.Register("Rancid", arena), nil, pool)
	fsF_Rancid.Func = Trapezoidal(0, 0, 1, 3)
	fsF_Delicious := NewFuzzySet(sym.Register("Delicious", arena), nil, pool)
	fsF_Delicious.Func = Trapezoidal(7, 9, 10, 10)
	
	vFood.AddTerm(fsF_Rancid.ID, fsF_Rancid)
	vFood.AddTerm(fsF_Delicious.ID, fsF_Delicious)

	// Tip terms (Universe 0-25)
	fsT_Cheap := NewFuzzySet(sym.Register("Cheap", arena), make([]float64, 26), pool)
	fsT_Average := NewFuzzySet(sym.Register("Average", arena), make([]float64, 26), pool)
	fsT_Generous := NewFuzzySet(sym.Register("Generous", arena), make([]float64, 26), pool)
	for i := 0; i <= 25; i++ {
		fsT_Cheap.Universe[i] = float64(i)
		fsT_Average.Universe[i] = float64(i)
		fsT_Generous.Universe[i] = float64(i)
	}
	fsT_Cheap.Func = Triangular(0, 5, 10)
	fsT_Average.Func = Triangular(10, 15, 20)
	fsT_Generous.Func = Triangular(20, 25, 30)
	// Sample
	for i := 0; i <= 25; i++ {
		fsT_Cheap.Members[i] = fsT_Cheap.Func(float64(i))
		fsT_Average.Members[i] = fsT_Average.Func(float64(i))
		fsT_Generous.Members[i] = fsT_Generous.Func(float64(i))
	}
	vTip.AddTerm(fsT_Cheap.ID, fsT_Cheap)
	vTip.AddTerm(fsT_Average.ID, fsT_Average)
	vTip.AddTerm(fsT_Generous.ID, fsT_Generous)

	engine.AddVariable(vService)
	engine.AddVariable(vFood)
	engine.AddVariable(vTip)

	// Rule 1: IF Service IS Poor OR Food IS Rancid THEN Tip IS Cheap
	r1 := NewFuzzyRule(1.0)
	r1.AddAntecedent(vService.ID, fsS_Poor.ID, false)
	r1.AddAntecedent(vFood.ID, fsF_Rancid.ID, false)
	// OR aggregation implicitly if we use another rule? No, in Mamdani rules antecedents are AND. 
	// To simulate OR, we split into two rules.
	r1.SetConsequent(vTip.ID, fsT_Cheap.ID)
	engine.AddRule(*r1)

	r1b := NewFuzzyRule(1.0)
	r1b.AddAntecedent(vFood.ID, fsF_Rancid.ID, false)
	r1b.SetConsequent(vTip.ID, fsT_Cheap.ID)
	engine.AddRule(*r1b)

	// Rule 2: IF Service IS Good THEN Tip IS Average
	r2 := NewFuzzyRule(1.0)
	r2.AddAntecedent(vService.ID, fsS_Good.ID, false)
	r2.SetConsequent(vTip.ID, fsT_Average.ID)
	engine.AddRule(*r2)

	// Rule 3: IF Service IS Excellent OR Food IS Delicious THEN Tip IS Generous
	r3 := NewFuzzyRule(1.0)
	r3.AddAntecedent(vService.ID, fsS_Excellent.ID, false)
	r3.SetConsequent(vTip.ID, fsT_Generous.ID)
	engine.AddRule(*r3)

	r3b := NewFuzzyRule(1.0)
	r3b.AddAntecedent(vFood.ID, fsF_Delicious.ID, false)
	r3b.SetConsequent(vTip.ID, fsT_Generous.ID)
	engine.AddRule(*r3b)

	return engine, sym
}

// Helper for HVAC problem (TSK)
func setupHVAC(pool *memory.Pool, arena *memory.Arena) (*TSKEngine, *SymbolTable) {
	sym := NewSymbolTable(arena)
	engine := NewTSKEngine()

	vTempErr := NewLinguisticVar(sym.Register("TempErr", arena))
	vRate := NewLinguisticVar(sym.Register("Rate", arena))
	
	vTempErr.AddTerm(1, &FuzzySet{ID: 1, Func: Gaussian(-5, 2)})
	vTempErr.AddTerm(2, &FuzzySet{ID: 2, Func: Gaussian(0, 2)})
	vTempErr.AddTerm(3, &FuzzySet{ID: 3, Func: Gaussian(5, 2)})

	vRate.AddTerm(1, &FuzzySet{ID: 1, Func: Gaussian(-2, 1)})
	vRate.AddTerm(2, &FuzzySet{ID: 2, Func: Gaussian(0, 1)})
	vRate.AddTerm(3, &FuzzySet{ID: 3, Func: Gaussian(2, 1)})

	engine.AddVariable(vTempErr)
	engine.AddVariable(vRate)

	// Create a few rules for fan speed
	// Consequent: Fan Speed (linear)
	r1 := TSKRule{
		Antecedents: []FuzzyCondition{{Variable: vTempErr.ID, Term: 1, Negated: false}},
		Consequent:  LinearTSK{Coeffs: map[VarID]float64{vTempErr.ID: 0.5}, Intercept: -2},
		Weight:      1.0,
	}
	engine.AddRule(r1)

	r2 := TSKRule{
		Antecedents: []FuzzyCondition{{Variable: vTempErr.ID, Term: 3, Negated: false}},
		Consequent:  LinearTSK{Coeffs: map[VarID]float64{vTempErr.ID: 0.5}, Intercept: 2},
		Weight:      1.0,
	}
	engine.AddRule(r2)

	return engine, sym
}

func TestIntegrationTipping(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	engine, sym := setupTipping(pool, arena)
	
	inputs := map[VarID]float64{
		sym.Register("Service", arena): 7.0, // Good to Excellent
		sym.Register("Food", arena):    8.0, // Delicious
	}

	start := time.Now()
	outSet, err := engine.Evaluate(inputs, pool)
	dur := time.Since(start)

	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	tip := Centroid(outSet)
	t.Logf("Computed Tip: %.2f%% in %v", tip, dur)

	// We expect tip to be roughly 15-22%
	if tip < 15.0 || tip > 25.0 {
		t.Errorf("Tip %.2f out of expected range", tip)
	}

	if dur > time.Millisecond {
		t.Errorf("Mamdani Evaluate took %v, expected < 1ms", dur)
	}
}

func TestIntegrationHVAC(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	engine, sym := setupHVAC(pool, arena)
	
	inputs := map[VarID]float64{
		sym.Register("TempErr", arena): 4.0, // Getting hot
		sym.Register("Rate", arena):    1.0, 
	}

	start := time.Now()
	fanSpeed, err := engine.Evaluate(inputs)
	dur := time.Since(start)

	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	t.Logf("Computed Fan Speed: %.2f in %v", fanSpeed, dur)

	if dur > time.Millisecond {
		t.Errorf("TSK Evaluate took %v, expected < 1ms", dur)
	}
}
