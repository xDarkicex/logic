package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func TestStutterAtom(t *testing.T) {
	p := Atom{ID: 0}
	if !IsStutterInvariant(p) {
		t.Error("atoms should be stutter-invariant")
	}
}

func TestStutterBoolean(t *testing.T) {
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p ∧ q
	if !IsStutterInvariant(And{Left: p, Right: q}) {
		t.Error("p∧q should be stutter-invariant")
	}
	// p → ¬q
	if !IsStutterInvariant(Implies{Antecedent: p, Consequent: Not{Formula: q}}) {
		t.Error("p→¬q should be stutter-invariant")
	}
}

func TestStutterBox(t *testing.T) {
	p := Atom{ID: 0}
	// □p
	if !IsStutterInvariant(Box{Formula: p, Rel: RelCausal}) {
		t.Error("□p should be stutter-invariant")
	}
	// □○p — Box absorbs Next sensitivity
	if !IsStutterInvariant(Box{Formula: Next{Formula: p}, Rel: RelCausal}) {
		t.Error("□○p should be stutter-invariant (□ absorbs ○)")
	}
}

func TestStutterDiamond(t *testing.T) {
	p := Atom{ID: 0}
	// ◇p
	if !IsStutterInvariant(Diamond{Formula: p, Rel: RelCausal}) {
		t.Error("◇p should be stutter-invariant")
	}
	// ◇○p — Diamond absorbs Next sensitivity
	if !IsStutterInvariant(Diamond{Formula: Next{Formula: p}, Rel: RelCausal}) {
		t.Error("◇○p should be stutter-invariant (◇ absorbs ○)")
	}
}

func TestStutterNext(t *testing.T) {
	p := Atom{ID: 0}
	// ○p — Next is NOT stutter-invariant at top level
	if IsStutterInvariant(Next{Formula: p}) {
		t.Error("○p should NOT be stutter-invariant")
	}
	// ○□p — Next outside modal operator
	if IsStutterInvariant(Next{Formula: Box{Formula: p, Rel: RelCausal}}) {
		t.Error("○□p should NOT be stutter-invariant")
	}
}

func TestStutterUntil(t *testing.T) {
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p U q
	if !IsStutterInvariant(Until{Left: p, Right: q}) {
		t.Error("pUq should be stutter-invariant")
	}
	// ○p U q — Next on left side
	if IsStutterInvariant(Until{Left: Next{Formula: p}, Right: q}) {
		t.Error("○p U q should NOT be stutter-invariant")
	}
}

func TestStutterComplex(t *testing.T) {
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// □(p ⇒ ◇(q ∧ ○r)) — ○ buried under ◇ under □ → stutter-invariant
	r := Atom{ID: 2}
	f := Box{
		Formula: Implies{
			Antecedent: p,
			Consequent: Diamond{
				Formula: And{Left: q, Right: Next{Formula: r}},
				Rel:     RelCausal,
			},
		},
		Rel: RelCausal,
	}
	if !IsStutterInvariant(f) {
		t.Error("□(p⇒◇(q∧○r)) should be stutter-invariant (○ under ◇ under □)")
	}
	// ○(p ⇒ □q) — Next at top
	f2 := Next{Formula: Implies{Antecedent: p, Consequent: Box{Formula: q, Rel: RelCausal}}}
	if IsStutterInvariant(f2) {
		t.Error("○(p⇒□q) should NOT be stutter-invariant (○ at top)")
	}
}

func TestCanCompress(t *testing.T) {
	p := Atom{ID: 0}
	if !CanCompress(Box{Formula: p, Rel: RelCausal}) {
		t.Error("□p should be compressible")
	}
	if CanCompress(Next{Formula: p}) {
		t.Error("○p should not be compressible")
	}
}

func TestStutterCompress(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	val := []TruthValueSlice{
		{1.0, 0.0},
		{1.0, 0.0}, // duplicate
		{1.0, 0.0}, // duplicate
		{0.0, 1.0},
		{0.0, 1.0}, // duplicate
		{1.0, 1.0},
	}
	compressed := StutterCompress(val, pool)
	if len(compressed) != 3 {
		t.Errorf("compress: expected 3, got %d", len(compressed))
	}
	ratio := CompressRatio(val, compressed)
	if ratio != 0.5 {
		t.Errorf("ratio: expected 0.5, got %v", ratio)
	}
}

func TestStutterCompressSingle(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	val := []TruthValueSlice{{1.0}}
	compressed := StutterCompress(val, pool)
	if len(compressed) != 1 {
		t.Errorf("single state: expected 1, got %d", len(compressed))
	}
}

func TestStutterCC(t *testing.T) {
	p := Atom{ID: 0}
	for i := 0; i < 100; i++ {
		IsStutterInvariant(p)
		IsStutterInvariant(Box{Formula: Next{Formula: p}, Rel: RelCausal})
		IsStutterInvariant(Next{Formula: Box{Formula: p, Rel: RelCausal}})
	}
}
