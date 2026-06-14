package xai

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

func TestSHAPExtractor(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	importances := map[fuzzy.VarID]float64{
		1: 0.8,
		2: 0.2,
		3: -0.5, // Should clamp weight to 0
		4: 1.5,  // Should clamp weight to 1
		5: 0.0,
	}

	engine := SHAPExtractor(importances, pool)

	// We expect 5 rules (one for each feature)
	// plus we want to make sure the output variable is registered.
	
	if engine == nil {
		t.Fatal("Expected engine, got nil")
	}
	
	// Evaluate with some inputs
	inputs := map[fuzzy.VarID]float64{
		1: 1.0, // High impact match
		2: 1.0, 
		3: 1.0,
		4: 1.0,
		5: 1.0,
	}

	outSet, err := engine.Evaluate(inputs, pool)
	if err != nil {
		t.Fatalf("Evaluation error: %v", err)
	}

	// Should not be nil
	if outSet == nil {
		t.Fatal("Expected output fuzzy set, got nil")
	}
}

func TestGNNExtractor(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	nodes := []fuzzy.VarID{1, 2, 3}
	edges := map[fuzzy.VarID][]fuzzy.VarID{
		1: {2},
		2: {3},
	}

	engine := GNNExtractor(nodes, edges, pool)

	if engine == nil {
		t.Fatal("Expected engine, got nil")
	}

	// Evaluate: Node 1 is active
	inputs := map[fuzzy.VarID]float64{
		1: 1.0, 
	}
	// We expect rule 1->2 to fire.
	// We don't have input for 2, so 2->3 rule will cause an error if we evaluate fully.
	// But let's just make sure it parses and sets up rules properly.
	
	// Add dummy variables for missing inputs just to pass evaluation
	inputs[2] = 0.0
	inputs[3] = 0.0

	// Actually GNNExtractor produces Mamdani rules where consequent is just setting a term.
	// Since engine evaluates all rules, if 1 is active, 2 becomes active. 
	_, err = engine.Evaluate(inputs, pool)
	if err != nil {
		t.Fatalf("Evaluation error: %v", err)
	}
}

func TestZeroDataFCM(t *testing.T) {
	concepts := []Concept{1, 2, 3}
	bounds := []Interval{
		{Min: 0, Max: 10},
		{Min: 5, Max: 5}, // Zero width
		{Min: -10, Max: 10},
	}

	fcm := ZeroDataFCM(concepts, bounds)

	if len(fcm.Concepts) != 3 {
		t.Errorf("Expected 3 concepts, got %d", len(fcm.Concepts))
	}

	// Test zero width
	if w := fcm.Weights[1][2]; w != 0.0 {
		t.Errorf("Expected 0 weight for zero width, got %v", w)
	}

	// Test self loop
	if w := fcm.Weights[1][1]; w != 0.0 {
		t.Errorf("Expected 0 weight for self loop, got %v", w)
	}

	// Test mismatch safetybounds length
	fcmBad := ZeroDataFCM(concepts, []Interval{})
	if len(fcmBad.Weights) != 0 {
		t.Error("Expected empty weights on mismatch")
	}

	// Test clipping
	boundsClip := []Interval{
		{Min: 0, Max: 2},   // center 1
		{Min: 100, Max: 102}, // center 101, width 2
	}
	fcmClip := ZeroDataFCM([]Concept{1, 2}, boundsClip)
	
	// w = (101 - 1) / 2 = 50 -> clipped to 1.0
	if w := fcmClip.Weights[1][2]; w != 1.0 {
		t.Errorf("Expected weight clipped to 1.0, got %v", w)
	}

	// Reverse clip
	// w = (1 - 101) / 2 = -50 -> clipped to -1.0
	if w := fcmClip.Weights[2][1]; w != -1.0 {
		t.Errorf("Expected weight clipped to -1.0, got %v", w)
	}
}
