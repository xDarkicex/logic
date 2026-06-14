package fuzzy

import (
	"math"
	"testing"

	"github.com/xDarkicex/memory"
)

func TestTruthValue(t *testing.T) {
	v := TruthValue(0.5)

	if err := v.Validate(); err != nil {
		t.Errorf("Expected valid TruthValue, got %v", err)
	}

	invalidLow := TruthValue(-0.1)
	if err := invalidLow.Validate(); err == nil {
		t.Error("Expected error for negative TruthValue")
	}

	invalidHigh := TruthValue(1.1)
	if err := invalidHigh.Validate(); err == nil {
		t.Error("Expected error for >1.0 TruthValue")
	}

	c := v.Complement()
	if c != 0.5 {
		t.Errorf("Expected complement 0.5, got %v", c)
	}

	v2 := TruthValue(0.75)
	if v2.Complement() != 0.25 {
		t.Errorf("Expected complement 0.25, got %v", v2.Complement())
	}

	v3 := TruthValue(0.3)
	// Testing Equals with floating point imprecision
	v4 := TruthValue(0.1 + 0.2) // typical float issue
	if !v3.Equals(v4) {
		t.Errorf("Expected %v to equal %v", v3, v4)
	}

	if v3.Equals(v2) {
		t.Errorf("Expected %v to not equal %v", v3, v2)
	}
}

func TestSymbolTable(t *testing.T) {
	arena, err := memory.NewArena(1024 * 1024)
	if err != nil {
		t.Fatalf("Failed to create arena: %v", err)
	}
	defer arena.Free()

	sym := NewSymbolTable(arena)
	if sym.Len() != 0 {
		t.Errorf("Expected len 0, got %v", sym.Len())
	}

	id1 := sym.Register("temperature", arena)
	if id1 != 1 {
		t.Errorf("Expected ID 1, got %v", id1)
	}
	if sym.Len() != 1 {
		t.Errorf("Expected len 1, got %v", sym.Len())
	}

	// Re-register should return the same ID
	id2 := sym.Register("temperature", arena)
	if id1 != id2 {
		t.Errorf("Expected ID %v, got %v", id1, id2)
	}

	id3 := sym.Register("humidity", arena)
	if id3 != 2 {
		t.Errorf("Expected ID 2, got %v", id3)
	}

	name1 := sym.Name(id1)
	if string(name1) != "temperature" {
		t.Errorf("Expected 'temperature', got '%s'", name1)
	}

	// Lookup missing
	if _, ok := sym.Lookup("pressure", 12345); ok {
		t.Error("Expected pressure to not be found")
	}

	// Out of bounds / invalid ID
	if sym.Name(0) != nil {
		t.Error("Expected nil for ID 0")
	}
	if sym.Name(999) != nil {
		t.Error("Expected nil for out of bounds ID")
	}
}

func TestSymbolTable_EnsureCapacity(t *testing.T) {
	arena, err := memory.NewArena(1024 * 1024)
	if err != nil {
		t.Fatalf("Failed to create arena: %v", err)
	}
	defer arena.Free()

	sym := NewSymbolTable(arena)
	// Force it to grow. The initial capacity is 128
	for i := 0; i < 150; i++ {
		sym.Register(string(rune(i+1000)), arena)
	}

	if sym.Len() != 150 {
		t.Errorf("Expected len 150, got %v", sym.Len())
	}
	if sym.capacity != 256 {
		t.Errorf("Expected capacity 256, got %v", sym.capacity)
	}
}

func TestFuzzySet(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	u := []float64{0, 10, 20, 30}
	fs := NewFuzzySet(1, u, pool)

	if fs.ID != 1 {
		t.Errorf("Expected ID 1, got %v", fs.ID)
	}
	if len(fs.Universe) != 4 {
		t.Errorf("Expected universe len 4, got %v", len(fs.Universe))
	}
	if len(fs.Members) != 4 {
		t.Errorf("Expected members len 4, got %v", len(fs.Members))
	}

	// Set members
	fs.Members[0] = 0.0
	fs.Members[1] = 0.5
	fs.Members[2] = 1.0
	fs.Members[3] = 0.0

	if fs.Membership(10) != 0.5 {
		t.Errorf("Expected membership 0.5 at x=10, got %v", fs.Membership(10))
	}
	// Miss
	if fs.Membership(15) != 0.0 {
		t.Errorf("Expected membership 0.0 at x=15, got %v", fs.Membership(15))
	}

	// Function evaluation
	fs2 := NewFuzzySet(2, nil, pool)
	fs2.Func = func(x float64) TruthValue {
		return TruthValue(math.Min(1.0, math.Max(0.0, x/100.0)))
	}
	if fs2.Membership(50) != 0.5 {
		t.Errorf("Expected func membership 0.5, got %v", fs2.Membership(50))
	}

	// Alpha cut
	cut := fs.AlphaCut(0.5)
	if len(cut) != 2 || cut[0] != 10 || cut[1] != 20 {
		t.Errorf("Expected alpha cut [10, 20], got %v", cut)
	}

	sup := fs.Support()
	if len(sup) != 2 || sup[0] != 10 || sup[1] != 20 {
		t.Errorf("Expected support [10, 20], got %v", sup)
	}

	core := fs.Core()
	if len(core) != 1 || core[0] != 20 {
		t.Errorf("Expected core [20], got %v", core)
	}

	if fs.Height() != 1.0 {
		t.Errorf("Expected height 1.0, got %v", fs.Height())
	}

	if !fs.IsNormal() {
		t.Error("Expected IsNormal true")
	}

	fs.Members[2] = 0.8
	if fs.IsNormal() {
		t.Error("Expected IsNormal false")
	}

	card := fs.Cardinality()
	if card != 1.3 { // 0.0 + 0.5 + 0.8 + 0.0
		t.Errorf("Expected cardinality 1.3, got %v", card)
	}
}

func TestLinguisticVar(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	lv := NewLinguisticVar(1, pool)
	if lv.ID != 1 {
		t.Errorf("Expected ID 1, got %v", lv.ID)
	}

	fs1 := NewFuzzySet(10, []float64{0, 10}, pool)
	fs1.Members[0] = 0.2
	fs1.Members[1] = 0.8

	fs2 := NewFuzzySet(20, []float64{0, 10}, pool)
	fs2.Members[0] = 0.7
	fs2.Members[1] = 0.3

	lv.AddTerm(10, fs1)
	lv.AddTerm(20, fs2)

	if lv.GetTerm(10) != fs1 {
		t.Error("Expected to retrieve fs1")
	}

	res := lv.Fuzzify(10)
	if len(res) != 2 {
		t.Errorf("Expected 2 fuzzified results, got %v", len(res))
	}
	if res[10] != 0.8 {
		t.Errorf("Expected fs1 membership 0.8, got %v", res[10])
	}
	if res[20] != 0.3 {
		t.Errorf("Expected fs2 membership 0.3, got %v", res[20])
	}
}

func TestFuzzyRule(t *testing.T) {
	rule := NewFuzzyRule(0.9)
	if rule.Weight != 0.9 {
		t.Errorf("Expected weight 0.9, got %v", rule.Weight)
	}

	rule.AddAntecedent(1, 10, true)
	if len(rule.Antecedents) != 1 {
		t.Errorf("Expected 1 antecedent, got %v", len(rule.Antecedents))
	}
	if rule.Antecedents[0].Variable != 1 || rule.Antecedents[0].Term != 10 || !rule.Antecedents[0].Negated {
		t.Errorf("Incorrect antecedent: %+v", rule.Antecedents[0])
	}

	rule.SetConsequent(2, 20)
	if rule.Consequent.Variable != 2 || rule.Consequent.Term != 20 || rule.Consequent.Negated {
		t.Errorf("Incorrect consequent: %+v", rule.Consequent)
	}
}

func TestSymbolTable_Collision(t *testing.T) {
	arena, err := memory.NewArena(1024 * 1024)
	if err != nil {
		t.Fatalf("Failed to create arena: %v", err)
	}
	defer arena.Free()

	sym := NewSymbolTable(arena)
	
	// Create an artificial collision by exploiting the lookup logic
	// We'll directly inject into names to test the collision resolution
	sym.names = memory.ArenaAppend(arena, sym.names, nameEntry{
		data: []byte("foo"),
		len:  3,
		hash: 123,
	})
	sym.names = memory.ArenaAppend(arena, sym.names, nameEntry{
		data: []byte("bar"),
		len:  3,
		hash: 123,
	})
	
	sym.byName[123] = 1 // points to "foo" (ID=1)
	
	// Lookup "foo"
	id, ok := sym.Lookup("foo", 123)
	if !ok || id != 1 {
		t.Errorf("Expected to find foo at ID 1, got %v, %v", id, ok)
	}
	
	// Lookup "bar" - should trigger collision loop
	id, ok = sym.Lookup("bar", 123)
	if !ok || id != 2 {
		t.Errorf("Expected to find bar at ID 2, got %v, %v", id, ok)
	}
}
