package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

func newPINSPool(t *testing.T) *memory.Pool {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	return pool
}

func TestNewDepMatrix(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(10, 64, pool)
	if m.rows != 10 {
		t.Errorf("rows: got %d, want 10", m.rows)
	}
	if m.wordsPerRow != 1 {
		t.Errorf("wordsPerRow: got %d, want 1 (64 bits)", m.wordsPerRow)
	}
}

func TestDepMatrixSetAndHas(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(4, 128, pool)

	m.Set(0, fuzzy.VarID(5))
	m.Set(0, fuzzy.VarID(100))
	m.Set(1, fuzzy.VarID(10))

	if !m.Has(0, fuzzy.VarID(5)) {
		t.Error("formula 0 should depend on atom 5")
	}
	if !m.Has(0, fuzzy.VarID(100)) {
		t.Error("formula 0 should depend on atom 100")
	}
	if !m.Has(1, fuzzy.VarID(10)) {
		t.Error("formula 1 should depend on atom 10")
	}
	if m.Has(0, fuzzy.VarID(10)) {
		t.Error("formula 0 should NOT depend on atom 10")
	}
	if m.Has(2, fuzzy.VarID(5)) {
		t.Error("formula 2 (empty) should NOT depend on atom 5")
	}
}

func TestDepMatrixOutOfBounds(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(2, 10, pool)
	// Setting beyond atom count should be no-op
	m.Set(0, fuzzy.VarID(99)) // > atomCount
	if m.Has(0, fuzzy.VarID(99)) {
		t.Error("out-of-bounds atom should not be set")
	}
}

func TestDepMatrixAnySet(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(2, 128, pool)
	m.Set(0, fuzzy.VarID(3))
	m.Set(0, fuzzy.VarID(70))

	// Dirty mask with atom 3 set
	dirty := []uint64{1 << 3, 0, 0} // word 0 bit 3
	if !m.AnySet(0, dirty) {
		t.Error("should detect dirty atom 3")
	}
	// Dirty mask with atom 70 set (word 1, bit 6)
	dirty = []uint64{0, 1 << 6, 0}
	if !m.AnySet(0, dirty) {
		t.Error("should detect dirty atom 70")
	}
	// Clean mask
	dirty = []uint64{0, 0, 0}
	if m.AnySet(0, dirty) {
		t.Error("clean mask should not trigger")
	}
}

func TestDepMatrixSpan(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(1, 128, pool)
	m.Set(0, fuzzy.VarID(10))
	m.Set(0, fuzzy.VarID(100))
	span := m.Span(0)
	// 100 - 10 + 1 = 91
	if span != 91 {
		t.Errorf("span: got %d, want 91", span)
	}
}

func TestDepMatrixSpanEmpty(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(1, 128, pool)
	if m.Span(0) != 0 {
		t.Errorf("empty row span: got %d, want 0", m.Span(0))
	}
}

func TestEventSpan(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(3, 128, pool)
	m.Set(0, fuzzy.VarID(0))
	m.Set(0, fuzzy.VarID(10))  // span = 11
	m.Set(1, fuzzy.VarID(50))
	m.Set(1, fuzzy.VarID(60))  // span = 11
	// row 2 empty, span = 0
	es := m.EventSpan()
	if es != 22 {
		t.Errorf("event span: got %d, want 22", es)
	}
}

func TestWorldDirtyMarkAndRow(t *testing.T) {
	pool := newPINSPool(t)
	d := NewWorldDirty(3, 128, pool)

	d.Mark(0, fuzzy.VarID(5))
	d.Mark(0, fuzzy.VarID(100))
	d.Mark(1, fuzzy.VarID(10))

	row0 := d.Row(0)
	if (row0[0] & (1 << 5)) == 0 {
		t.Error("world 0 should have atom 5 dirty")
	}
	row1 := d.Row(1)
	if (row1[0] & (1 << 10)) == 0 {
		t.Error("world 1 should have atom 10 dirty")
	}
}

func TestWorldDirtyMarkAllAndClean(t *testing.T) {
	pool := newPINSPool(t)
	d := NewWorldDirty(2, 128, pool)

	d.MarkAll(0)
	row := d.Row(0)
	if row[0] != ^uint64(0) {
		t.Error("MarkAll should set all bits")
	}
	d.Clean(0)
	row = d.Row(0)
	if row[0] != 0 {
		t.Error("Clean should clear all bits")
	}
}

func TestComputeDeps(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(4, 128, pool)

	// p && (q || r) at row 0,  (p -> q) at row 1
	reg := NewRegistry(64, pool)
	a := reg.Intern(Atom{ID: fuzzy.VarID(1)})
	b := reg.Intern(Atom{ID: fuzzy.VarID(2)})
	c := reg.Intern(Atom{ID: fuzzy.VarID(3)})
	f1 := reg.Intern(And{Left: a, Right: Or{Left: b, Right: c}})
	f2 := reg.Intern(Implies{Antecedent: a, Consequent: b})

	ComputeDeps(f1, m, 0)
	ComputeDeps(f2, m, 1)

	// f1 should depend on atoms 1,2,3
	if !m.Has(0, fuzzy.VarID(1)) || !m.Has(0, fuzzy.VarID(2)) || !m.Has(0, fuzzy.VarID(3)) {
		t.Error("f1 should depend on atoms 1,2,3")
	}
	// f2 should depend only on atoms 1,2
	if !m.Has(1, fuzzy.VarID(1)) || !m.Has(1, fuzzy.VarID(2)) {
		t.Error("f2 should depend on atoms 1,2")
	}
	if m.Has(1, fuzzy.VarID(3)) {
		t.Error("f2 should NOT depend on atom 3")
	}
}

func TestPINSRegistryIntern(t *testing.T) {
	pool := newPINSPool(t)
	pr := NewPINSRegistry(64, 64, 128, pool)

	p := pr.Intern(Atom{ID: fuzzy.VarID(1)})
	q := pr.Intern(Atom{ID: fuzzy.VarID(2)})
	and := pr.Intern(And{Left: p, Right: q})

	id := pr.LookupID(and)
	if id < 0 {
		t.Fatal("expected valid ID for And")
	}
	if !pr.DepMatrix().Has(id, fuzzy.VarID(1)) {
		t.Error("And should depend on atom 1")
	}
	if !pr.DepMatrix().Has(id, fuzzy.VarID(2)) {
		t.Error("And should depend on atom 2")
	}
}

func TestPINSRegistryIsClean(t *testing.T) {
	pool := newPINSPool(t)
	pr := NewPINSRegistry(64, 64, 128, pool)

	p := pr.Intern(Atom{ID: fuzzy.VarID(1)})
	id := pr.LookupID(p)
	if id < 0 {
		t.Fatal("expected valid ID")
	}

	dirty := NewWorldDirty(2, 128, pool)

	// No atoms dirty → clean
	if !pr.IsClean(id, 0, dirty) {
		t.Error("formula should be clean when no atoms dirty")
	}

	// Mark atom 1 dirty → formula depends on atom 1 → not clean
	dirty.Mark(0, fuzzy.VarID(1))
	if pr.IsClean(id, 0, dirty) {
		t.Error("formula should be dirty when atom 1 is marked")
	}

	// Mark atom 2 dirty → formula does NOT depend on 2 → still dirty from atom 1
	// But clean at world 1 where nothing is marked
	if !pr.IsClean(id, 1, dirty) {
		t.Error("formula should be clean at world 1")
	}
}

func TestPINSRegistryDeepNesting(t *testing.T) {
	pool := newPINSPool(t)
	pr := NewPINSRegistry(128, 128, 200, pool)

	// Build: □(p → ◇(q && r))
	pp := pr.Intern(Atom{ID: fuzzy.VarID(1)})
	qq := pr.Intern(Atom{ID: fuzzy.VarID(2)})
	rr := pr.Intern(Atom{ID: fuzzy.VarID(3)})
	inner := pr.Intern(And{Left: qq, Right: rr})
	dia := pr.Intern(Diamond{Formula: inner, Rel: RelCausal})
	imp := pr.Intern(Implies{Antecedent: pp, Consequent: dia})
	box := pr.Intern(Box{Formula: imp, Rel: RelCausal})

	id := pr.LookupID(box)
	if id < 0 {
		t.Fatal("expected valid ID")
	}

	m := pr.DepMatrix()
	if !m.Has(id, fuzzy.VarID(1)) || !m.Has(id, fuzzy.VarID(2)) || !m.Has(id, fuzzy.VarID(3)) {
		t.Error("□(p→◇(q&&r)) should depend on atoms 1,2,3")
	}
}

func TestPINSRegistryIncrementalSkip(t *testing.T) {
	pool := newPINSPool(t)
	pr := NewPINSRegistry(64, 64, 128, pool)

	// Register several formulas
	f1 := pr.Intern(Atom{ID: fuzzy.VarID(1)})
	f2 := pr.Intern(Atom{ID: fuzzy.VarID(50)})
	f3 := pr.Intern(And{Left: pr.Intern(Atom{ID: fuzzy.VarID(10)}), Right: pr.Intern(Atom{ID: fuzzy.VarID(20)})})

	id1 := pr.LookupID(f1)
	id2 := pr.LookupID(f2)
	id3 := pr.LookupID(f3)

	dirty := NewWorldDirty(1, 128, pool)
	dirty.Mark(0, fuzzy.VarID(1))

	// f1 depends on 1 → dirty. f2 depends on 50 → clean. f3 depends on 10,20 → clean.
	if pr.IsClean(id1, 0, dirty) {
		t.Error("f1 should be dirty (depends on atom 1)")
	}
	if !pr.IsClean(id2, 0, dirty) {
		t.Error("f2 should be clean (depends on atom 50, not 1)")
	}
	if !pr.IsClean(id3, 0, dirty) {
		t.Error("f3 should be clean (depends on atoms 10,20)")
	}
}

func TestDepMatrixSpanOptimized(t *testing.T) {
	pool := newPINSPool(t)
	m := NewDepMatrix(1, 200, pool)
	// Atoms clustered at positions 50 and 55
	m.Set(0, fuzzy.VarID(50))
	m.Set(0, fuzzy.VarID(55))
	span := m.Span(0)
	if span != 6 {
		t.Errorf("clustered span: got %d, want 6", span)
	}
}
