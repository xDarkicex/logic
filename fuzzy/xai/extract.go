package xai

import (
	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// OutputVarID is the designated output variable for extraction.
const OutputVarID fuzzy.VarID = 9999

// SHAPExtractor converts feature importance scores (like those from SHAP)
// into a transparent Fuzzy Rule Base. Features with higher importance
// receive heavier rule weights and tighter membership constraints.
func SHAPExtractor(importances map[fuzzy.VarID]float64, pool *memory.Pool) *fuzzy.MamdaniEngine {
	engine := fuzzy.NewMamdaniEngine(pool)
	
	// Create output variable
	outVar := fuzzy.NewLinguisticVar(OutputVarID, pool)
	outFsHigh := fuzzy.NewFuzzySet(1, nil, pool)
	outFsHigh.Func = fuzzy.Gaussian(1.0, 0.2) // "High Impact"
	outVar.AddTerm(1, outFsHigh)
	engine.AddVariable(outVar)

	// Create a rule for each feature
	for vID, imp := range importances {
		// Create input linguistic variable
		inVar := fuzzy.NewLinguisticVar(vID, pool)
		
		// Create "High" term based on importance.
		// Higher importance -> narrower sigma (more specific rule)
		sigma := 0.5
		if imp > 0 {
			sigma = 0.5 / imp
		}
		
		termFs := fuzzy.NewFuzzySet(1, nil, pool)
		termFs.Func = fuzzy.Gaussian(1.0, sigma)
		inVar.AddTerm(1, termFs)
		engine.AddVariable(inVar)

		// Rule: IF vID IS High THEN Output IS HighImpact (Weight = importance)
		weight := fuzzy.TruthValue(imp)
		if weight > 1.0 {
			weight = 1.0
		} else if weight < 0.0 {
			weight = 0.0
		}

		rule := fuzzy.NewFuzzyRule(weight)
		rule.AddAntecedent(vID, 1, false)
		rule.SetConsequent(OutputVarID, 1)
		engine.AddRule(*rule)
	}

	return engine
}

// GNNExtractor constructs relational fuzzy rules from a graph structure.
// Nodes are linguistic variables, edges represent conditional dependencies.
// Useful for creating interpretable rule bases from knowledge graphs.
func GNNExtractor(nodes []fuzzy.VarID, edges map[fuzzy.VarID][]fuzzy.VarID, pool *memory.Pool) *fuzzy.MamdaniEngine {
	engine := fuzzy.NewMamdaniEngine(pool)

	// Register all nodes as variables with a default "Active" term
	for _, nodeID := range nodes {
		v := fuzzy.NewLinguisticVar(nodeID, pool)
		fs := fuzzy.NewFuzzySet(1, nil, pool)
		fs.Func = fuzzy.Singleton(1.0) // "Active"
		v.AddTerm(1, fs)
		engine.AddVariable(v)
	}

	// For each edge A -> B, create rule: IF A IS Active THEN B IS Active
	for srcID, targets := range edges {
		for _, tgtID := range targets {
			rule := fuzzy.NewFuzzyRule(1.0)
			rule.AddAntecedent(srcID, 1, false)
			rule.SetConsequent(tgtID, 1)
			engine.AddRule(*rule)
		}
	}

	return engine
}
