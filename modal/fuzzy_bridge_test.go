package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type bridgeFixtures struct {
	pool  *memory.Pool
	arena *memory.Arena
}

func newBridgeFixtures(t *testing.T) *bridgeFixtures {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	arena, err := memory.NewArena(1024 * 1024)
	if err != nil {
		t.Fatalf("arena: %v", err)
	}
	t.Cleanup(func() {
		pool.Reset()
		arena.Free()
	})
	return &bridgeFixtures{pool: pool, arena: arena}
}

func setupWeightedFrame(t *testing.T, fx *bridgeFixtures) (*Frame, *Model) {
	t.Helper()
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddWeightedRelation(w0, w1, RelCausal, 0.7)
	frame.AddWeightedRelation(w0, w2, RelCausal, 0.4)
	frame.AddRelation(w0, w2, RelAssociation) // unweighted

	model := NewModel(frame, 1, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 1.0)
	model.SetTruth(w1, fuzzy.VarID(0), 0.8)
	model.SetTruth(w2, fuzzy.VarID(0), 0.3)
	return frame, model
}

func TestAddWeightedRelation(t *testing.T) {
	fx := newBridgeFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddWeightedRelation(w0, w1, RelCausal, 0.5)

	if frame.WeightedEdgeCount() != 1 {
		t.Errorf("expected 1 weighted edge, got %d", frame.WeightedEdgeCount())
	}

	targets := frame.WeightedAccessible(w0, RelCausal)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Weight != 0.5 {
		t.Errorf("expected weight 0.5, got %v", targets[0].Weight)
	}
}

func TestWeightedAccessibleMixed(t *testing.T) {
	fx := newBridgeFixtures(t)
	frame, _ := setupWeightedFrame(t, fx)

	// RelCausal: has 2 weighted edges
	targets := frame.WeightedAccessible(0, RelCausal)
	if len(targets) != 2 {
		t.Fatalf("expected 2 weighted targets for RelCausal, got %d", len(targets))
	}

	// RelAssociation: has 1 unweighted edge → weight=1.0
	targets = frame.WeightedAccessible(0, RelAssociation)
	if len(targets) != 1 {
		t.Fatalf("expected 1 target for RelAssociation, got %d", len(targets))
	}
	if targets[0].Weight != 1.0 {
		t.Errorf("unweighted edge should have weight 1.0, got %v", targets[0].Weight)
	}

	// RelProcedural: no edges
	targets = frame.WeightedAccessible(0, RelProcedural)
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestWeightedFrameAccessibility(t *testing.T) {
	fx := newBridgeFixtures(t)
	frame, _ := setupWeightedFrame(t, fx)

	w := WeightedFrameAccessibility(0, 1, RelCausal, frame)
	if w != 0.7 {
		t.Errorf("weighted edge: got %v, want 0.7", w)
	}

	w = WeightedFrameAccessibility(0, 2, RelAssociation, frame)
	if w != 1.0 {
		t.Errorf("unweighted edge: got %v, want 1.0", w)
	}

	w = WeightedFrameAccessibility(1, 0, RelCausal, frame)
	if w != 0.0 {
		t.Errorf("no edge: got %v, want 0.0", w)
	}
}

func TestBoxFuzzy(t *testing.T) {
	fx := newBridgeFixtures(t)
	_, model := setupWeightedFrame(t, fx)
	cfg := WeightedFuzzyConfig()

	// □P at w0: accessible w1(w=0.7,p=0.8) and w2(w=0.4,p=0.3)
	// imp(0.7, 0.8) = min(1, 1-0.7+0.8) = min(1, 1.1) = 1.0
	// imp(0.4, 0.3) = min(1, 1-0.4+0.3) = min(1, 0.9) = 0.9
	// tnorm(1.0, tnorm(1.0, 1.0)) = productTNorm(productTNorm(1.0, 1.0), 0.9) = 0.9
	// Wait — tnorm is applied iteratively: result = tnorm(result, imp)
	// result starts at 1.0
	// Step 1: result = productTNorm(1.0, 1.0) = 1.0
	// Step 2: result = productTNorm(1.0, 0.9) = 0.9
	tv, err := BoxFuzzy(Atom{ID: fuzzy.VarID(0)}, 0, model, cfg, RelCausal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approxEqualF(tv, 0.9) {
		t.Errorf("BoxFuzzy weighted: got %v, want 0.9", tv)
	}
}

func TestBoxFuzzyEmpty(t *testing.T) {
	fx := newBridgeFixtures(t)
	_, model := setupWeightedFrame(t, fx)
	cfg := DefaultFuzzyConfig()

	tv, err := BoxFuzzy(Atom{ID: fuzzy.VarID(0)}, 0, model, cfg, RelProcedural)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("BoxFuzzy with no accessible: got %v, want 1.0", tv)
	}
}

func TestDiamondFuzzy(t *testing.T) {
	fx := newBridgeFixtures(t)
	_, model := setupWeightedFrame(t, fx)
	cfg := WeightedFuzzyConfig()

	// ◇P at w0: accessible w1(w=0.7,p=0.8) and w2(w=0.4,p=0.3)
	// Step 1: comb = tnorm(0.7, 0.8) = 0.56, result = tconorm(0.0, 0.56) = 0.56
	// Step 2: comb = tnorm(0.4, 0.3) = 0.12, result = tconorm(0.56, 0.12) = probSum(0.56, 0.12)
	//   = 0.56 + 0.12 - 0.56*0.12 = 0.68 - 0.0672 = 0.6128
	tv, err := DiamondFuzzy(Atom{ID: fuzzy.VarID(0)}, 0, model, cfg, RelCausal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := TruthValue(0.56 + 0.12 - 0.56*0.12)
	if !approxEqualF(tv, expected) {
		t.Errorf("DiamondFuzzy weighted: got %v, want ~%v", tv, expected)
	}
}

func TestDiamondFuzzyEmpty(t *testing.T) {
	fx := newBridgeFixtures(t)
	_, model := setupWeightedFrame(t, fx)
	cfg := DefaultFuzzyConfig()

	tv, err := DiamondFuzzy(Atom{ID: fuzzy.VarID(0)}, 0, model, cfg, RelProcedural)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("DiamondFuzzy with no accessible: got %v, want 0.0", tv)
	}
}

func TestBoxFuzzyDefaultConfig(t *testing.T) {
	fx := newBridgeFixtures(t)
	_, model := setupWeightedFrame(t, fx)
	cfg := DefaultFuzzyConfig()

	// Default: Godel min/max, Lukasiewicz implication
	// Same weighted case but with Godel t-norm (min)
	// Step 1: imp(0.7, 0.8) = min(1, 1-0.7+0.8) = 1.0, result = min(1.0, 1.0) = 1.0
	// Step 2: imp(0.4, 0.3) = min(1, 1-0.4+0.3) = 0.9, result = min(1.0, 0.9) = 0.9
	tv, err := BoxFuzzy(Atom{ID: fuzzy.VarID(0)}, 0, model, cfg, RelCausal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approxEqualF(tv, 0.9) {
		t.Errorf("BoxFuzzy default: got %v, want 0.9", tv)
	}
}

func TestFuzzyEntailment(t *testing.T) {
	fx := newBridgeFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddWeightedRelation(w0, w1, RelCausal, 0.8)

	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 1.0)
	model.SetTruth(w0, fuzzy.VarID(1), 1.0)
	model.SetTruth(w1, fuzzy.VarID(0), 1.0)
	model.SetTruth(w1, fuzzy.VarID(1), 0.0)

	// P∧Q → P should be high entailment
	premises := []Formula{And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}}
	conclusion := Atom{ID: fuzzy.VarID(0)}

	tv, err := FuzzyEntailment(premises, conclusion, model, WeightedFuzzyConfig(), RelCausal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv < 0.5 {
		t.Errorf("P∧Q → P entailment should be high: got %v", tv)
	}
}

func TestFuzzyEntailmentEmptyPremises(t *testing.T) {
	fx := newBridgeFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	tv, err := FuzzyEntailment(nil, Atom{ID: fuzzy.VarID(0)}, model, DefaultFuzzyConfig(), RelCausal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("empty premises: got %v, want 0.0", tv)
	}
}

func TestDefaultFuzzyConfig(t *testing.T) {
	cfg := DefaultFuzzyConfig()
	// Verify functions are set
	if cfg.TNorm(0.3, 0.7) != 0.3 {
		t.Error("default TNorm should be min")
	}
	if cfg.TConorm(0.3, 0.7) != 0.7 {
		t.Error("default TConorm should be max")
	}
	imp := float64(cfg.Implication(0.7, 0.3))
	if imp < 0.599 || imp > 0.601 {
		t.Errorf("Łukasiewicz(0.7,0.3) = %v, want ~0.6", imp)
	}
}

func approxEqualF(a, b TruthValue) bool {
	d := float64(a) - float64(b)
	if d < 0 {
		d = -d
	}
	return d < 1e-9
}

func TestWeightedFuzzyConfig(t *testing.T) {
	cfg := WeightedFuzzyConfig()
	// Product t-norm
	if cfg.TNorm(0.3, 0.7) != 0.21 {
		t.Errorf("product t-norm: got %v, want 0.21", cfg.TNorm(0.3, 0.7))
	}
	// Probabilistic sum t-conorm
	expected := TruthValue(0.3 + 0.7 - 0.3*0.7)
	if cfg.TConorm(0.3, 0.7) != expected {
		t.Errorf("probabilistic sum: got %v, want %v", cfg.TConorm(0.3, 0.7), expected)
	}
}
