package modal

import (
	"testing"

	"github.com/xDarkicex/gobdd"
	"github.com/xDarkicex/memory"
)

func newDecomposer(t *testing.T) *Decomposer {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	ctx := NewBDDCtx(8, pool)
	return NewDecomposer(ctx, pool)
}

func TestDecomposeAtom(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	dc := d.Decompose(p)
	if len(dc.Now) != 1 {
		t.Errorf("atom: expected 1 NOW, got %d", len(dc.Now))
	}
	if dc.HasNext() || dc.HasPromise() {
		t.Error("atom should be trivial (no NEXT or PROMISE)")
	}
	if !dc.IsTrivial() {
		t.Error("atom decomposition should be trivial")
	}
}

func TestDecomposeBox(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	// □p — Boolean inside Box → NOW
	b := Box{Formula: p, Rel: RelCausal}
	dc := d.Decompose(b)
	if len(dc.Now) != 1 {
		t.Errorf("□p: expected 1 NOW, got %d", len(dc.Now))
	}
}

func TestDecomposeBoxNonBoolean(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	// □◇p — Diamond inside Box → non-Boolean → NEXT
	inner := Diamond{Formula: p, Rel: RelCausal}
	b := Box{Formula: inner, Rel: RelCausal}
	dc := d.Decompose(b)
	if len(dc.Next) != 1 {
		t.Errorf("□◇p: expected 1 NEXT, got %d", len(dc.Next))
	}
}

func TestDecomposeDiamond(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	// ◇p — Boolean inside → NOW
	dd := Diamond{Formula: p, Rel: RelCausal}
	dc := d.Decompose(dd)
	if len(dc.Now) != 1 {
		t.Errorf("◇p: expected 1 NOW, got %d", len(dc.Now))
	}
}

func TestDecomposeDiamondBox(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	// ◇□p — Box inside Diamond → PROMISE
	inner := Box{Formula: p, Rel: RelCausal}
	dd := Diamond{Formula: inner, Rel: RelCausal}
	dc := d.Decompose(dd)
	if len(dc.Promise) != 1 {
		t.Errorf("◇□p: expected 1 PROMISE, got %d", len(dc.Promise))
	}
}

func TestDecomposeNext(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	n := Next{Formula: p}
	dc := d.Decompose(n)
	if len(dc.Next) != 1 {
		t.Errorf("○p: expected 1 NEXT, got %d", len(dc.Next))
	}
}

func TestDecomposeUntil(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	u := Until{Left: p, Right: q}
	dc := d.Decompose(u)
	if len(dc.Promise) != 1 {
		t.Errorf("pUq: expected 1 PROMISE, got %d", len(dc.Promise))
	}
}

func TestDecomposeComplex(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// (□p) ∧ ○(q) ∧ (p U q)
	f := And{
		Left: Box{Formula: p, Rel: RelCausal},
		Right: And{
			Left:  Next{Formula: q},
			Right: Until{Left: p, Right: q},
		},
	}
	dc := d.Decompose(f)
	if len(dc.Now) < 1 {
		t.Errorf("complex: expected at least 1 NOW, got %d", len(dc.Now))
	}
	if len(dc.Next) < 1 {
		t.Errorf("complex: expected at least 1 NEXT, got %d", len(dc.Next))
	}
	if len(dc.Promise) < 1 {
		t.Errorf("complex: expected at least 1 PROMISE, got %d", len(dc.Promise))
	}
}

func TestIsPureBoolean(t *testing.T) {
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	if !isPureBoolean(p) {
		t.Error("atom is pure Boolean")
	}
	if !isPureBoolean(Not{Formula: p}) {
		t.Error("¬p is pure Boolean")
	}
	if !isPureBoolean(And{Left: p, Right: q}) {
		t.Error("p∧q is pure Boolean")
	}
	if isPureBoolean(Box{Formula: p, Rel: RelCausal}) {
		t.Error("□p is not pure Boolean")
	}
}

func TestBuildNowBDD(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	now := []Formula{p, q}
	node := d.BuildNowBDD(now)
	if node == gobdd.False || node == gobdd.True {
		t.Error("NOW BDD should be non-terminal for p∧q")
	}
	// Empty NOW → True
	node2 := d.BuildNowBDD(nil)
	if node2 != gobdd.True {
		t.Error("empty NOW = True")
	}
}

func TestCCDecomposer(t *testing.T) {
	d := newDecomposer(t)
	p := Atom{ID: 0}
	for i := 0; i < 100; i++ {
		dc := d.Decompose(Box{Formula: p, Rel: RelCausal})
		if !dc.IsTrivial() {
			t.Error("□p should be trivial")
		}
	}
}
