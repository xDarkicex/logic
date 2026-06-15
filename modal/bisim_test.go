package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newBisimContractor(t *testing.T) *BisimContractor {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	ctx := NewBDDCtx(16, pool)
	return NewBisimContractor(ctx, pool)
}

func newTestModel(t *testing.T, pool *memory.Pool) *Model {
	t.Helper()
	arena, _ := memory.NewArena(1024*1024)
	t.Cleanup(func(){ arena.Free() })
	frame := NewFrame(pool, arena)
	// 3 worlds, 1 variable. w0: true, w1: true, w2: false.
	// w0 and w1 have same valuation → bisimilar if accessibility matches.
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	// w0 → w2, w1 → w2 (same accessibility pattern)
	frame.edges = append(frame.edges, Edge{Src: w0, Dst: w2, Rel: RelCausal})
	frame.edges = append(frame.edges, Edge{Src: w1, Dst: w2, Rel: RelCausal})
	return &Model{
		frame: frame,
		valuation: []TruthValueSlice{
			{1.0}, // w0
			{1.0}, // w1 — same as w0
			{0.0}, // w2
		},
	}
}

func TestBisimContractIdentical(t *testing.T) {
	bc := newBisimContractor(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	m := newTestModel(t, pool)

	reduced := bc.Contract(m)
	// w0 and w1 are bisimilar → should be merged into ≤ 2 worlds.
	if len(reduced.valuation) == 3 {
		t.Error("bisimilar worlds w0 and w1 should be merged")
	}
	if len(reduced.valuation) > 2 {
		t.Errorf("expected ≤ 2 worlds after contraction, got %d", len(reduced.valuation))
	}
}

func TestBisimContractDistinct(t *testing.T) {
	bc := newBisimContractor(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024*1024)
	defer func(){ arena.Free() }()
	frame := NewFrame(pool, arena)
	frame.AddWorld()
	frame.AddWorld()
	// Different valuations → not bisimilar → not merged.
	m := &Model{
		frame: frame,
		valuation: []TruthValueSlice{
			{1.0}, // w0
			{0.0}, // w1
		},
	}
	reduced := bc.Contract(m)
	if len(reduced.valuation) != 2 {
		t.Errorf("distinct worlds should not merge: got %d", len(reduced.valuation))
	}
}

func TestBisimSingleWorld(t *testing.T) {
	bc := newBisimContractor(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024*1024)
	defer func(){ arena.Free() }()
	frame := NewFrame(pool, arena)
	frame.AddWorld()
	m := &Model{
		frame:     frame,
		valuation: []TruthValueSlice{{1.0}},
	}
	reduced := bc.Contract(m)
	if len(reduced.valuation) != 1 {
		t.Error("single-world model should stay single")
	}
}

func TestBisimWorldSignature(t *testing.T) {
	bc := newBisimContractor(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	m := newTestModel(t, pool)
	classes := []int32{0, 0, 1} // w0 and w1 in class 0, w2 in class 1
	vars := bc.denseClassVars(classes, int32(len(classes)))
	sig0 := bc.worldSignature(m, 0, classes, vars)
	sig1 := bc.worldSignature(m, 1, classes, vars)
	// w0 and w1 have identical accessibility → identical BDD signatures.
	if sig0 != sig1 {
		t.Error("w0 and w1 should have identical BDD signatures")
	}
	sig2 := bc.worldSignature(m, 2, classes, vars)
	if sig0 == sig2 {
		t.Error("w2 should have different signature from w0")
	}
}

func TestBisimCC(t *testing.T) {
	bc := newBisimContractor(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	m := newTestModel(t, pool)
	for i := 0; i < 10; i++ {
		bc.Contract(m)
	}
}
