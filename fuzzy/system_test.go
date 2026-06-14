package fuzzy

import (
	"testing"

	"github.com/xDarkicex/logic/core"
	"github.com/xDarkicex/memory"
)

func TestSystemBasic(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()
	sym := NewSymbolTable(arena)

	engine := NewMamdaniEngine(pool)
	v := NewLinguisticVar(sym.Register("Temp", arena), pool)
	fs := NewFuzzySet(sym.Register("Hot", arena), nil, pool)
	fs.Func = Gaussian(100.0, 10.0) // Peak at 100
	v.AddTerm(sym.Register("Hot", arena), fs)
	engine.AddVariable(v)

	sys := NewFuzzyLogicSystem(engine, sym, pool, arena)

	if sys.Name() != "fuzzy" {
		t.Errorf("Expected name 'fuzzy', got %s", sys.Name())
	}

	ops := sys.SupportedOperators()
	if len(ops) != 3 {
		t.Errorf("Expected 3 operators, got %d", len(ops))
	}

	// Validate valid condition
	err := sys.Validate("Temp IS Hot")
	if err != nil {
		t.Errorf("Unexpected validate error: %v", err)
	}

	// Validate invalid condition
	err = sys.Validate("IF Temp IS Hot") // "IF" is not handled by ParseCondition
	if err == nil {
		t.Error("Expected validate error for invalid condition")
	}

	// Evaluate condition
	ctx := core.NewEvaluationContext()
	ctx.Set("Temp", 100.0) // Peak match (1.0)
	
	res, err := sys.Evaluate("Temp IS Hot", ctx)
	if err != nil {
		t.Errorf("Evaluate error: %v", err)
	}
	if !res {
		t.Error("Expected Evaluate to return true")
	}

	// Evaluate threshold
	ctx.Set("Temp", 0.0) // 0 match
	res, err = sys.Evaluate("Temp IS Hot", ctx)
	if err != nil {
		t.Errorf("Evaluate error: %v", err)
	}
	if res {
		t.Error("Expected Evaluate to return false")
	}

	// Evaluate errors
	_, err = sys.Evaluate("Bad IS Hot", ctx) // Variable not in engine
	if err == nil {
		t.Error("Expected error for missing engine variable")
	}

	// Context missing value
	ctx2 := core.NewEvaluationContext()
	_, err = sys.Evaluate("Temp IS Hot", ctx2)
	if err == nil {
		t.Error("Expected error for missing context input")
	}

	// Parse failure
	_, err = sys.Evaluate("Temp IS", ctx)
	if err == nil {
		t.Error("Expected error for parse failure")
	}

	// Empty condition (trick the parser or just not possible through string directly)
	// No string triggers empty condition because ParseCondition expects at least one IDENT.
}
