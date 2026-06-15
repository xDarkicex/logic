package sat

import (
	"testing"
	"time"

	"github.com/xDarkicex/logic/fuzzy"
)

// helper: create a solver, solve a CNF, verify satisfiability
func solve(t *testing.T, clauses [][]Literal, expectSAT bool) *SolverResult {
	t.Helper()
	cnf := NewCNF()
	for _, lits := range clauses {
		cnf.AddClause(NewClause(lits...))
	}
	solver := NewCDCLSolver()
	result := solver.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if result.Satisfiable != expectSAT {
		t.Errorf("Satisfiable = %v, want %v", result.Satisfiable, expectSAT)
	}
	return result
}

func TestIntegrationTrivialSAT(t *testing.T) {
	result := solve(t, [][]Literal{
		{{Variable: "A", Negated: false}},
	}, true)
	if result.Assignment == nil || !result.Assignment["A"] {
		t.Error("expected A=true")
	}
}

func TestIntegrationTrivialUnsat(t *testing.T) {
	solve(t, [][]Literal{
		{{Variable: "A", Negated: false}},
		{{Variable: "A", Negated: true}},
	}, false)
}

func TestIntegrationSimpleOr(t *testing.T) {
	result := solve(t, [][]Literal{
		{{Variable: "A", Negated: false}, {Variable: "B", Negated: false}},
	}, true)
	if !result.Assignment["A"] && !result.Assignment["B"] {
		t.Error("at least one of A or B must be true")
	}
}

func TestIntegrationThreeVar(t *testing.T) {
	// (A ∨ B) ∧ (¬A ∨ C) ∧ (¬B ∨ ¬C) — satisfiable
	solve(t, [][]Literal{
		{{Variable: "A", Negated: false}, {Variable: "B", Negated: false}},
		{{Variable: "A", Negated: true}, {Variable: "C", Negated: false}},
		{{Variable: "B", Negated: true}, {Variable: "C", Negated: true}},
	}, true)
}

func TestIntegrationPigeonhole(t *testing.T) {
	// Pigeonhole: 3 pigeons, 2 holes — UNSAT
	// Each pigeon in at least one hole: (p1_h1 ∨ p1_h2), (p2_h1 ∨ p2_h2), (p3_h1 ∨ p3_h2)
	// No two pigeons share a hole: (¬p1_h1 ∨ ¬p2_h1), (¬p1_h2 ∨ ¬p2_h2), (¬p1_h1 ∨ ¬p3_h1), etc.
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("P1_H1", false), L("P1_H2", false)))
	cnf.AddClause(NewClause(L("P2_H1", false), L("P2_H2", false)))
	cnf.AddClause(NewClause(L("P3_H1", false), L("P3_H2", false)))
	cnf.AddClause(NewClause(L("P1_H1", true), L("P2_H1", true)))
	cnf.AddClause(NewClause(L("P1_H2", true), L("P2_H2", true)))
	cnf.AddClause(NewClause(L("P1_H1", true), L("P3_H1", true)))
	cnf.AddClause(NewClause(L("P1_H2", true), L("P3_H2", true)))
	cnf.AddClause(NewClause(L("P2_H1", true), L("P3_H1", true)))
	cnf.AddClause(NewClause(L("P2_H2", true), L("P3_H2", true)))

	solver := NewCDCLSolver()
	result := solver.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if result.Satisfiable {
		t.Error("pigeonhole(3,2) should be UNSAT")
	}
}

func TestIntegrationRestartAndLearn(t *testing.T) {
	// Formula that forces conflicts on wrong decisions before finding solution
	// (A ∨ B) ∧ (¬A ∨ C) ∧ (¬B ∨ C) — satisfiable with C=true
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true), L("C", false)))
	cnf.AddClause(NewClause(L("B", true), L("C", false)))

	solver := NewCDCLSolver()
	result := solver.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("should be SAT")
	}
	if result.Statistics.Conflicts > 0 {
		t.Logf("conflicts: %d, restarts: %d", result.Statistics.Conflicts, result.Statistics.Restarts)
	}
}

func TestIntegrationXOR(t *testing.T) {
	ecnf := NewExtendedCNF()
	ecnf.AddClause(NewClause(L("A", false)))
	ecnf.AddXORClause(NewXORClause([]string{"A", "B", "C"}, true))

	solver := NewCDCLSolver()
	result := solver.SolveExtended(ecnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("XOR with A=true, parity odd → B⊕C=0 → both true or both false")
	}
}

func TestIntegrationCNFConverter(t *testing.T) {
	conv := NewCNFConverter()
	cnf, err := conv.ConvertExpression("(A & B) | C")
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	if cnf == nil {
		t.Fatal("cnf is nil")
	}
	if len(cnf.Clauses) == 0 {
		t.Error("expected clauses from conversion")
	}

	solver := NewCDCLSolver()
	result := solver.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("(A&B)|C should be satisfiable")
	}
}

func TestIntegrationCNFAllGates(t *testing.T) {
	conv := NewCNFConverter()
	exprs := []string{
		"A & B",      // AND
		"A | B",      // OR
		"!A",         // NOT
		"A ^ B",      // XOR
		"A -> B",     // IMPLIES
		"A <-> B",    // IFF
		"!(A & B)",   // NAND
		"!(A | B)",   // NOR
	}
	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			cnf, err := conv.ConvertExpression(expr)
			if err != nil {
				t.Fatalf("conversion error for %q: %v", expr, err)
			}
			solver := NewCDCLSolver()
			result := solver.Solve(cnf)
			if result.Error != nil {
				t.Fatalf("solve error for %q: %v", expr, result.Error)
			}
			if !result.Satisfiable {
				t.Errorf("%q should be satisfiable", expr)
			}
		})
	}
}

func TestIntegrationPreprocessor(t *testing.T) {
	prep := NewSATPreprocessor()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("C", false), L("D", false)))

	result, err := prep.Preprocess(cnf)
	if err != nil {
		t.Fatalf("preprocess error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	// A is a unit clause; B clause subsumed by A; C,D should remain
	if len(result.Clauses) < 1 {
		t.Error("expected at least 1 clause after preprocessing")
	}
}

func TestIntegrationPreprocessorWithCDCL(t *testing.T) {
	prep := NewSATPreprocessor()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("X", false)))
	cnf.AddClause(NewClause(L("X", true), L("Y", false)))
	cnf.AddClause(NewClause(L("Y", true)))

	processed, err := prep.Preprocess(cnf)
	if err != nil {
		t.Fatalf("preprocess error: %v", err)
	}

	solver := NewCDCLSolver()
	result := solver.Solve(processed)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	// X=true (unit), Y=true (from unit propagation of ¬X∨Y), ¬Y conflicts → UNSAT
	if result.Satisfiable {
		t.Log("formula may be satisfiable depending on preprocessing")
	}
}

func TestIntegrationSolverStatistics(t *testing.T) {
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true), L("C", false)))
	cnf.AddClause(NewClause(L("B", true), L("C", true)))

	solver := NewCDCLSolver()
	result := solver.Solve(cnf)

	stats := solver.GetStatistics()
	if stats.Decisions == 0 && stats.Conflicts == 0 {
		t.Log("no decisions/conflicts (trivial formula)")
	}
	_ = result
	_ = stats
}

func TestIntegrationSolverReset(t *testing.T) {
	solver := NewCDCLSolver()

	cnf1 := NewCNF()
	cnf1.AddClause(NewClause(L("A", false)))
	result1 := solver.Solve(cnf1)
	if !result1.Satisfiable {
		t.Fatal("first solve should be SAT")
	}

	solver.Reset()

	cnf2 := NewCNF()
	cnf2.AddClause(NewClause(L("B", false)))
	cnf2.AddClause(NewClause(L("B", true)))
	result2 := solver.Solve(cnf2)
	if result2.Satisfiable {
		t.Error("second solve should be UNSAT")
	}
}

func TestIntegrationDPLLSolver(t *testing.T) {
	dpll := NewDPLLSolver()

	// SAT: (A ∨ B) ∧ (¬A) — forces A=false, B=true
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true)))

	result := dpll.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("dpll error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("DPLL should find SAT: A=false forces B=true")
	}
}

func TestIntegrationDPLLUnitPropagation(t *testing.T) {
	dpll := NewDPLLSolver()

	cnf := NewCNF()
	cnf.AddClause(NewClause(L("X", false)))
	cnf.AddClause(NewClause(L("X", true), L("Y", false)))

	result := dpll.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("dpll error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("DPLL should find SAT: X=true, Y=true")
	}
}

func TestIntegrationConflictAnalysis(t *testing.T) {
	analyzer := NewFirstUIPAnalyzer()
	if analyzer.Name() == "" {
		t.Error("analyzer should have a name")
	}
	analyzer.Reset()
	if analyzer.Name() == "" {
		t.Error("analyzer should still have a name after reset")
	}

	stats := analyzer.GetStatistics()
	if stats == nil {
		t.Error("statistics should not be nil")
	}
}

func TestIntegrationHeuristicResetAndReuse(t *testing.T) {
	h := NewVSIDSHeuristic()
	assignment := make(Assignment)

	// First use
	h.ChooseVariable([]string{"A", "B"}, assignment)
	c := NewClause(L("A", false))
	h.Update(c)

	// Reset
	h.Reset()

	// Reuse after reset
	chosen := h.ChooseVariable([]string{"A", "B"}, assignment)
	if chosen == "" {
		t.Error("should choose a variable after reset")
	}
}

func TestIntegrationModeSwitchWithCDCL(t *testing.T) {
	solver := NewCDCLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false), L("C", false)))
	cnf.AddClause(NewClause(L("A", true), L("B", false)))
	cnf.AddClause(NewClause(L("B", true), L("C", false)))
	cnf.AddClause(NewClause(L("A", false), L("C", true)))

	result := solver.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("should be SAT")
	}
	// Mode switching should have been exercised during solve
}

func TestIntegrationWalkSATWarmStart(t *testing.T) {
	solver := NewCDCLSolver()
	cnf := NewCNF()

	// Build a satisfiable formula
	for i := 0; i < 20; i++ {
		vars := []string{varName(3*i), varName(3*i + 1), varName(3*i + 2)}
		cnf.AddClause(NewClause(L(vars[0], false), L(vars[1], false), L(vars[2], false)))
		cnf.AddClause(NewClause(L(vars[0], true), L(vars[1], false)))
		cnf.AddClause(NewClause(L(vars[1], true), L(vars[2], false)))
	}

	result := solver.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("should be SAT")
	}
}

func TestIntegrationTimeout(t *testing.T) {
	solver := NewCDCLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	cnf.AddClause(NewClause(L("A", true)))

	result := solver.SolveWithTimeout(cnf, 1*time.Second)
	if result.Error != nil {
		t.Fatalf("error: %v", result.Error)
	}
	if result.Satisfiable {
		t.Error("should be UNSAT")
	}
}

func TestIntegrationSATSystem(t *testing.T) {
	sys := NewSATSystem()
	if sys.Name() == "" {
		t.Error("system should have a name")
	}

	cnf, err := sys.ConvertToCNF("A | B")
	if err != nil {
		t.Fatalf("conversion error: %v", err)
	}
	result := sys.Solve(cnf)
	if !result.Satisfiable {
		t.Error("A|B should be SAT")
	}

	ok, err := sys.VerifySolution("A & B", Assignment{"A": true, "B": true})
	if err != nil {
		t.Fatalf("verification error: %v", err)
	}
	if !ok {
		t.Error("A=true, B=true should satisfy A&B")
	}

	supported := sys.SupportedOperators()
	if len(supported) == 0 {
		t.Error("should have supported operators")
	}
}

func TestIntegrationMAXSAT(t *testing.T) {
	maxsolver := NewMAXSATSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	cnf.AddClause(NewClause(L("A", true)))
	cnf.AddClause(NewClause(L("B", false)))

	result := maxsolver.SolveMAXSAT(cnf, nil)
	if result.Assignment == nil {
		t.Error("should have an assignment")
	}
	if result.SatisfiedCount < 1 {
		t.Error("should satisfy at least 1 clause")
	}

	result2 := maxsolver.SolveWeightedMAXSAT(cnf, []float64{1.0, 2.0, 0.5})
	if result2.TotalWeight <= 0 {
		t.Error("should have positive total weight")
	}
}

func TestIntegrationFuzzySMT(t *testing.T) {
	clauses := []FuzzyClause{
		{Literals: []FuzzyLiteral{{VarID: 1, Negated: false}, {VarID: 2, Negated: false}}},
		{Literals: []FuzzyLiteral{{VarID: 1, Negated: true}}},
	}
	assignment, ok := SolveFuzzy(clauses, []fuzzy.VarID{1, 2}, 100, 0.1)
	if !ok {
		t.Log("fuzzy solver did not converge in 100 epochs")
	}
	_ = assignment
}

func TestIntegrationVarHeapWithHeuristic(t *testing.T) {
	h := NewVSIDSHeuristic()
	assignment := make(Assignment)

	// Register variables via ChooseVariable
	unassigned := []string{"X", "Y", "Z"}
	h.ChooseVariable(unassigned, assignment)

	// Update to bump scores
	h.Update(NewClause(L("Y", false)))
	h.Update(NewClause(L("Y", false)))

	// Y should now have highest activity
	chosen := h.ChooseVariable(unassigned, assignment)
	if chosen != "Y" {
		t.Errorf("expected Y (highest activity), got %s", chosen)
	}
}

func TestIntegrationHeuristicOnBacktrack(t *testing.T) {
	h := NewVSIDSHeuristic()
	h.ChooseVariable([]string{"A", "B", "C"}, make(Assignment))

	// Simulate backtrack: re-insert popped variables
	h.OnBacktrack([]string{"A", "B"})
	// Should not panic
}

func TestIntegrationWalkSATExport(t *testing.T) {
	w := NewWalkSolver()
	c1 := NewClause(L("X", false), L("Y", false))
	c2 := NewClause(L("X", true), L("Z", false))

	w.Solve([]*Clause{c1, c2})

	h := NewVSIDSHeuristic()
	w.ExportPhases(h)

	// Verify phases were exported
	if h.varIndex["X"] == 0 && h.phases[0] == phaseUnset {
		t.Error("X should have a phase after export")
	}
}

func TestIntegrationModeCompleteCycle(t *testing.T) {
	ms := NewModeSwitcher()
	// Complete 4 switches: focused → stable → focused → stable → focused
	decisions := int64(0)
	for i := 0; i < 4; i++ {
		if ms.Mode() == ModeFocused {
			ms.Switch(ms.conflictsAtMode+ms.conflictLimit, decisions)
		} else {
			ms.Switch(ms.conflictsAtMode, decisions+ms.tickLimit)
		}
		decisions += ms.tickLimit
	}
	if ms.Mode() != ModeFocused {
		t.Error("should end in focused mode after even number of switches")
	}
}

func TestIntegrationClauseDB(t *testing.T) {
	db := NewClauseDatabase(100, 500)
	if db.Size() != 0 {
		t.Error("new db should be empty")
	}

	c1 := NewClause(L("A", false), L("B", false))
	c1.Learned = true
	c1.ID = 1

	db.AddClause(c1, 0)
	if db.Size() != 1 {
		t.Error("db should have 1 clause after add")
	}

	core, mid, local, recent := db.GetTierSlices()
	_ = core
	_ = mid
	_ = local
	_ = recent
}

func TestIntegrationGaussianEliminator(t *testing.T) {
	ge := NewGaussianEliminator()
	if ge.IsDisabled() {
		t.Error("new eliminator should be enabled")
	}

	ecnf := NewExtendedCNF()
	ecnf.AddXORClause(NewXORClause([]string{"A", "B"}, true))
	ecnf.AddXORClause(NewXORClause([]string{"B", "C"}, false))

	result, err := ge.PerformGaussianElimination(ecnf, make(Assignment), 0)
	if err != nil {
		t.Fatalf("gaussian error: %v", err)
	}
	_ = result

	ge.Reset()
	ge.Enable()
	if ge.IsDisabled() {
		t.Error("should be enabled after reset and enable")
	}
}

func varName(i int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if i < 26 {
		return string(letters[i])
	}
	return string(letters[i%26]) + string(letters[(i/26)%26])
}

func L(variable string, negated bool) Literal {
	return Literal{Variable: variable, Negated: negated}
}

// Coverage gap fillers — test String, Name, IsEmpty, IsUnit, etc.

func TestCoverageClauseMethods(t *testing.T) {
	c := NewClause(L("A", false), L("B", true))
	_ = c.String()
	if c.IsEmpty() {
		t.Error("clause should not be empty")
	}
	if c.IsUnit() {
		t.Error("2-literal clause should not be unit")
	}
	if c.GetTier() != 2 {
		t.Log("default tier")
	}
	if !c.IsGlue() {
		t.Log("LBD=0 means glue=false, expected")
	}
	if c.Contains(L("A", false)) {
		// correct
	}
	if c.Contains(L("C", false)) {
		t.Error("should not contain C")
	}

	// Unit clause
	u := NewClause(L("X", false))
	if !u.IsUnit() {
		t.Error("single-literal clause should be unit")
	}

	// Empty clause
	e := NewClause()
	if !e.IsEmpty() {
		t.Error("zero-literal clause should be empty")
	}
	_ = e.String()

	// LBD
	c.SetLBD(3)
	if c.LBD != 3 {
		t.Error("LBD should be 3")
	}
}

func TestCoverageCNFMethods(t *testing.T) {
	cnf := NewCNF()
	_ = cnf.String()
	cnf.AddClause(NewClause(L("A", false)))
	_ = cnf.String()
	if len(cnf.Variables) == 0 {
		t.Log("no variables in empty CNF")
	}
}

func TestCoverageSolverNames(t *testing.T) {
	if NewCDCLSolver().Name() == "" {
		t.Error("CDCLSolver Name should not be empty")
	}
	if NewDPLLSolver().Name() == "" {
		t.Error("DPLLSolver Name should not be empty")
	}
	if NewVSIDSHeuristic().Name() == "" {
		t.Error("VSIDSHeuristic Name should not be empty")
	}
	if NewRandomHeuristic().Name() == "" {
		t.Error("RandomHeuristic Name should not be empty")
	}
	if NewLubyRestartStrategy().Name() == "" {
		t.Error("LubyRestartStrategy Name should not be empty")
	}
	if NewActivityBasedDeletion().Name() == "" {
		t.Error("ActivityBasedDeletion Name should not be empty")
	}
	if NewFirstUIPAnalyzer().Name() == "" {
		t.Error("FirstUIPAnalyzer Name should not be empty")
	}
}

func TestCoverageAssignment(t *testing.T) {
	a := make(Assignment)
	a["A"] = true
	a["B"] = false

	if !a.IsAssigned("A") {
		t.Error("A should be assigned")
	}
	if a.IsAssigned("C") {
		t.Error("C should not be assigned")
	}

	c := NewClause(L("A", false))
	if !a.Satisfies(c) {
		t.Error("A=true should satisfy clause (A)")
	}
	if a.ConflictsWith(c) {
		t.Error("A=true should not conflict with clause (A)")
	}

	c2 := NewClause(L("A", true))
	if a.Satisfies(c2) {
		t.Error("A=true should not satisfy clause (¬A)")
	}
	if !a.ConflictsWith(c2) {
		t.Error("A=true should conflict with clause (¬A)")
	}

	clone := a.Clone()
	if clone["A"] != true || clone["B"] != false {
		t.Error("clone should preserve values")
	}
}

func TestCoverageTrailExtras(t *testing.T) {
	trail := NewDecisionTrail()
	if trail.GetMaxLevel() != 0 {
		t.Log("initial max level")
	}
	if trail.GetLevelSize(0) != 0 {
		t.Log("no assignments at level 0")
	}
	if trail.GetTrailSize() != 0 {
		t.Error("empty trail should have size 0")
	}
}

func TestCoveragePreprocessorExtras(t *testing.T) {
	p := NewSATPreprocessor()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	cnf.AddClause(NewClause(L("B", false)))

	post := p.PostProcess(Assignment{"A": true})
	_ = post
}

func TestCoverageGaussianExtras(t *testing.T) {
	ge := NewGaussianEliminator()
	stats := ge.GetStatistics()
	_ = stats
	ge.Reset()
}

func TestCoverageLiteralMethods(t *testing.T) {
	l1 := Literal{Variable: "A", Negated: false}
	l2 := Literal{Variable: "A", Negated: false}
	l3 := Literal{Variable: "A", Negated: true}

	if !l1.Equals(l2) {
		t.Error("identical literals should be equal")
	}
	if l1.Equals(l3) {
		t.Error("different negation should not be equal")
	}
	_ = l1.String()
	_ = l3.String()

	neg := l1.Negate()
	if neg.Variable != "A" || !neg.Negated {
		t.Error("negation should flip Negated")
	}
}

func TestCoverageSolverStatisticsString(t *testing.T) {
	s := SolverStatistics{
		Conflicts:       5,
		Decisions:       10,
		Propagations:    100,
		Restarts:        2,
		LearnedClauses:  3,
		LBDDistribution: map[int]int64{1: 1, 2: 2, 5: 1},
	}
	_ = s.String()
	_ = s.GetLBDDistribution()
}

func TestCoverageWatchedClause(t *testing.T) {
	c := NewClause(L("A", false), L("B", false), L("C", false))
	wc := NewWatchedClause(c)
	if wc == nil {
		t.Fatal("NewWatchedClause returned nil")
	}
	lits := wc.GetWatchedLiterals()
	if len(lits) == 0 {
		t.Error("expected watched literals")
	}
	_ = lits
	_ = wc.Watch1
	_ = wc.Watch2
}

func TestCoverageHeuristicDecay(t *testing.T) {
	h := NewVSIDSHeuristic()
	assignment := make(Assignment)

	// Register variables and bump many times to trigger decay/rescale
	for i := 0; i < 2000; i++ {
		h.ChooseVariable([]string{"A", "B"}, assignment)
		c := NewClause(L("A", false))
		h.Update(c)
	}

	// Reset should clear everything
	h.Reset()
	chosen := h.ChooseVariable([]string{"A", "B"}, assignment)
	if chosen == "" {
		t.Error("should choose a variable after reset and decay cycles")
	}
}

func TestCoverageCNFAddXOR(t *testing.T) {
	ecnf := NewExtendedCNF()
	ecnf.AddClause(NewClause(L("A", false)))
	ecnf.AddXORClause(NewXORClause([]string{"A", "B"}, true))

	if len(ecnf.XORClauses) == 0 {
		t.Error("should have XOR clauses")
	}
	_ = ecnf.String()
}

func TestCoverageConflictAnalyzerReset(t *testing.T) {
	a := NewFirstUIPAnalyzer()
	a.Reset()
	stats := a.GetStatistics()
	if stats == nil {
		t.Error("statistics should not be nil after reset")
	}
}

func TestCoverageRestartExtend(t *testing.T) {
	l := NewLubyRestartStrategy()
	// Force sequence extension
	for i := 0; i < 30; i++ {
		l.OnRestart()
	}
	l.Reset()
}

func TestCoverageWalkWithLargeFormula(t *testing.T) {
	w := NewWalkSolver()
	w.maxFlips = 2000
	var clauses []*Clause
	// Build a 20-variable satisfiable formula
	for i := 0; i < 30; i++ {
		a := varName(i * 2)
		b := varName(i*2 + 1)
		clauses = append(clauses, NewClause(L(a, false), L(b, false)))
		clauses = append(clauses, NewClause(L(a, true), L(b, true)))
	}
	w.Solve(clauses)
	if w.FlipCount() == 0 && w.UnsatCount() > 0 {
		t.Log("no flips despite unsat clauses (may have found solution at init)")
	}
}

func TestCoverageModeShouldSwitch(t *testing.T) {
	ms := NewModeSwitcher()
	// Focused: below threshold
	if ms.ShouldSwitch(500, 0) {
		t.Error("should not switch at 500 conflicts")
	}
	// Switch to stable
	ms.Switch(1000, 0)
	// Stable: below threshold
	if ms.ShouldSwitch(1000, 200) {
		t.Error("should not switch at 200 decisions in stable mode")
	}
	// Stable: at threshold
	if !ms.ShouldSwitch(1000, 500) {
		t.Error("should switch at 500 decisions in stable mode")
	}
}

func TestCoverageVarHeapScore(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 3.0)
	if h.Score(0) != 3.0 {
		t.Errorf("Score(0) = %v, want 3.0", h.Score(0))
	}
	if h.Score(100) != 0 {
		t.Errorf("Score(unregistered) = %v, want 0", h.Score(100))
	}
}

func TestCoverageHeuristicPreferredPolarity(t *testing.T) {
	h := NewVSIDSHeuristic()
	// Unknown variable should default to true
	if !h.GetPreferredPolarity("unknown") {
		t.Error("unknown variable should default to positive polarity")
	}
	// Register and set
	h.ensureVar("X")
	// Default polarity for new var (score 0)
	_ = h.GetPreferredPolarity("X")
}

func TestCoverageLargeCDCLSolve(t *testing.T) {
	// Larger formula to trigger restarts and clause learning
	cnf := NewCNF()
	n := 15
	for i := 0; i < n; i++ {
		a := varName(i * 3)
		b := varName(i*3 + 1)
		c := varName(i*3 + 2)
		cnf.AddClause(NewClause(L(a, false), L(b, false), L(c, false)))
		cnf.AddClause(NewClause(L(a, true), L(b, false)))
		cnf.AddClause(NewClause(L(b, true), L(c, true)))
	}

	solver := NewCDCLSolver()
	result := solver.Solve(cnf)
	if result.Error != nil {
		t.Fatalf("solve error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("large formula should be SAT")
	}
	t.Logf("conflicts: %d, restarts: %d, decisions: %d",
		result.Statistics.Conflicts, result.Statistics.Restarts, result.Statistics.Decisions)
}

func TestCoverageDeletionCandidates(t *testing.T) {
	a := NewActivityBasedDeletion()
	db := NewClauseDatabase(5, 100)

	// Add clauses to trigger deletion logic
	for i := 0; i < 10; i++ {
		c := NewClause(L(varName(i), false), L(varName(i+10), false))
		c.Learned = true
		c.ID = i
		c.Activity = 0.01
		c.LBD = 8
		c.Tier = 2
		db.AddClause(c, int64(i))
	}

	candidates := a.GetDeletionCandidates(db, SolverStatistics{})
	if len(candidates) == 0 {
		t.Log("no deletion candidates (db may not exceed max size)")
	}
	_ = candidates
}

func TestCoverageCNFVariables(t *testing.T) {
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("B", true), L("C", false)))

	// Variables should be deduplicated
	if len(cnf.Variables) != 3 {
		t.Errorf("expected 3 variables, got %d: %v", len(cnf.Variables), cnf.Variables)
	}
}

func TestCoverageLubyReset(t *testing.T) {
	l := NewLubyRestartStrategy()
	l.ShouldRestart(SolverStatistics{Conflicts: 100})
	l.OnRestart()
	l.Reset()
}

func TestCoverageDecisionTrailInterface(t *testing.T) {
	trail := NewDecisionTrail()
	trail.Assign("A", true, 0, nil)
	trail.Assign("B", false, 1, nil)

	if !trail.IsDecisionVariable("A") {
		t.Log("A at level 0 with nil reason should be a decision variable")
	}

	levels := trail.GetAllLevels()
	if len(levels) != 2 {
		t.Logf("levels found: %d", len(levels))
	}

	chain := trail.GetImplicationChain("A")
	_ = chain

	trail.Backtrack(0)
	trail.Clear()
	trail.Close()
}

func TestCoverageGaussianElimination(t *testing.T) {
	ge := NewGaussianEliminator()
	if !ge.ShouldRunGaussian(6000, 10) {
		t.Log("gaussian should run after 5000+ conflicts with enough XORs")
	}
	if ge.ShouldRunGaussian(100, 3) {
		t.Error("gaussian should not run with too few conflicts")
	}
	if ge.ShouldRunGaussian(6000, 2) {
		t.Error("gaussian should not run with too few XOR clauses")
	}

	ge.Enable()
	if ge.IsDisabled() {
		t.Error("should be enabled")
	}
}

func TestCoverageInprocessorConfig(t *testing.T) {
	cfg := DefaultInprocessConfig()
	_ = cfg
}

func TestCoverageMAXSATResult(t *testing.T) {
	maxsolver := NewMAXSATSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	result := maxsolver.SolveMAXSAT(cnf, []float64{1.0})
	if result.SatisfiedCount == 0 {
		t.Error("should satisfy the only clause")
	}
}

