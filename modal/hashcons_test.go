package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

func newHashConsPool(t *testing.T) *memory.Pool {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	return pool
}

func TestNewRegistry(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)
	if reg.count != 0 {
		t.Errorf("new registry count: got %d, want 0", reg.count)
	}
}

func TestInternAtom(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	a1 := Atom{ID: 1}
	a2 := Atom{ID: 1}
	a3 := Atom{ID: 2}

	c1 := reg.Intern(a1)
	c2 := reg.Intern(a2)
	c3 := reg.Intern(a3)

	if c1 != c2 {
		t.Error("identical atoms should return same pointer")
	}
	if c1 == c3 {
		t.Error("different atoms should return different pointers")
	}
}

func TestInternNot(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	a := reg.Intern(Atom{ID: 1})
	n1 := Not{Formula: a}
	n2 := Not{Formula: a}

	c1 := reg.Intern(n1)
	c2 := reg.Intern(n2)

	if c1 != c2 {
		t.Error("identical Not should return same pointer")
	}
}

func TestInternAnd(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	p := reg.Intern(Atom{ID: 1})
	q := reg.Intern(Atom{ID: 2})

	a1 := And{Left: p, Right: q}
	a2 := And{Left: p, Right: q}

	c1 := reg.Intern(a1)
	c2 := reg.Intern(a2)

	if c1 != c2 {
		t.Error("identical And should return same pointer")
	}
}

func TestInternAndCommutative(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	p := reg.Intern(Atom{ID: 1})
	q := reg.Intern(Atom{ID: 2})

	c1 := reg.Intern(And{Left: p, Right: q})
	c2 := reg.Intern(And{Left: q, Right: p})

	// NOT commutative — q && p has different hash and should be different
	if c1 == c2 {
		t.Error("p&&q should not equal q&&p (non-commutative by default)")
	}
}

func TestInternDeepTree(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	// Build And(Not(Atom(1)), Atom(2)) twice
	build := func() Formula {
		a := reg.Intern(Atom{ID: 1})
		n := reg.Intern(Not{Formula: a})
		b := reg.Intern(Atom{ID: 2})
		return reg.Intern(And{Left: n, Right: b})
	}

	t1 := build()
	t2 := build()

	if t1 != t2 {
		t.Error("structurally identical trees should return same pointer")
	}
}

func TestInternBox(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	p := reg.Intern(Atom{ID: 1})
	b1 := reg.Intern(Box{Formula: p, Rel: RelCausal})
	b2 := reg.Intern(Box{Formula: p, Rel: RelCausal})
	b3 := reg.Intern(Box{Formula: p, Rel: RelProcedural})

	if b1 != b2 {
		t.Error("identical Box should return same pointer")
	}
	if b1 == b3 {
		t.Error("Box with different Rel should return different pointers")
	}
}

func TestInternDiamond(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	p := reg.Intern(Atom{ID: 1})
	d1 := reg.Intern(Diamond{Formula: p, Rel: RelCausal})
	d2 := reg.Intern(Diamond{Formula: p, Rel: RelCausal})

	if d1 != d2 {
		t.Error("identical Diamond should return same pointer")
	}
}

func TestInternImplies(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	p := reg.Intern(Atom{ID: 1})
	q := reg.Intern(Atom{ID: 2})

	i1 := reg.Intern(Implies{Antecedent: p, Consequent: q})
	i2 := reg.Intern(Implies{Antecedent: p, Consequent: q})

	if i1 != i2 {
		t.Error("identical Implies should return same pointer")
	}
}

func TestInternIff(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	p := reg.Intern(Atom{ID: 1})
	q := reg.Intern(Atom{ID: 2})

	i1 := reg.Intern(Iff{Left: p, Right: q})
	i2 := reg.Intern(Iff{Left: p, Right: q})

	if i1 != i2 {
		t.Error("identical Iff should return same pointer")
	}
}

func TestGrowRehashes(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(4, pool) // tiny initial size forces growth

	// Insert many atoms — should trigger grow and rehash
	var prev Formula
	for i := 0; i < 100; i++ {
		f := reg.Intern(Atom{ID: fuzzy.VarID(i)})
		if prev != nil && f == prev {
			t.Errorf("different atoms should be distinct at %d", i)
		}
		prev = f
	}

	// All atoms should still be retrievable
	a50 := reg.Intern(Atom{ID: 50})
	a50Again := reg.Intern(Atom{ID: 50})
	if a50 != a50Again {
		t.Error("after grow, same atom should still return same pointer")
	}
}

func TestNilIntern(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(16, pool)

	if reg.Intern(nil) != nil {
		t.Error("Intern(nil) should return nil")
	}
}

func TestParserWithRegistry(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(64, pool)

	// Parse the same expression twice — should share sub-formula pointers
	parse := func() Formula {
		lexer := NewLexer("p && p", pool)
		tokens := lexer.Lex()
		parser := NewParser(tokens, "p && p")
		parser.SetRegistry(reg)
		f, err := parser.Parse()
		if err != nil {
			t.Fatalf("parse error: %v", err)
		}
		return f
	}

	f1 := parse()
	f2 := parse()

	if f1 != f2 {
		t.Error("same parsed expression should return same pointer from registry")
	}

	// Left and right of the And should also be the same pointer
	a := f1.(And)
	if a.Left != a.Right {
		t.Error("p && p should share the same atom pointer for p")
	}
}

func TestParserRegistryDeepStructure(t *testing.T) {
	pool := newHashConsPool(t)
	reg := NewRegistry(128, pool)

	parse := func(expr string) Formula {
		lexer := NewLexer(expr, pool)
		tokens := lexer.Lex()
		parser := NewParser(tokens, expr)
		parser.SetRegistry(reg)
		f, err := parser.Parse()
		if err != nil {
			t.Fatalf("parse(%q): %v", expr, err)
		}
		return f
	}

	expr1 := parse("[](p -> q)")
	expr2 := parse("[](p -> q)")

	if expr1 != expr2 {
		t.Error("identical parsed formulas should share pointer")
	}
}
