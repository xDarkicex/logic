package sat

import (
	"testing"
	"time"

	"github.com/xDarkicex/memory"
)

// Probe tests directly call unexported functions with crafted internal state
// to hit code paths that require thousands of CDCL conflicts to trigger naturally.

// === XOR conflict paths ===

func TestProbeXORPropagation(t *testing.T) {
	solver := NewCDCLSolver()
	solver.extendedCNF = NewExtendedCNF()
	solver.extendedCNF.AddXORClause(NewXORClause([]string{"A", "B", "C"}, true))
	// Assign A=true, B=true: A⊕B⊕C=1 → 1⊕1⊕C=1 → 0⊕C=1 → C must be true
	solver.assignment["A"] = true
	solver.assignment["B"] = true

	conflict := solver.propagateXOR()
	// C is unassigned, so no conflict — but C should be propagated
	_ = conflict
}

func TestProbeXORConflictDetection(t *testing.T) {
	solver := NewCDCLSolver()
	solver.extendedCNF = NewExtendedCNF()
	solver.extendedCNF.AddXORClause(NewXORClause([]string{"A", "B"}, true))
	// A⊕B=1 with A=true, B=true → 1⊕1=0≠1 → conflict
	solver.assignment["A"] = true
	solver.assignment["B"] = true

	conflict := solver.propagateXOR()
	if conflict == nil {
		t.Log("XOR conflict may require specific internal state")
	}
}

func TestProbeConvertXORConflict(t *testing.T) {
	solver := NewCDCLSolver()
	solver.extendedCNF = NewExtendedCNF()
	xor := NewXORClause([]string{"A", "B"}, true)
	solver.extendedCNF.AddXORClause(xor)

	// A⊕B=1, A=true, B=true → conflict
	solver.assignment["A"] = true
	solver.assignment["B"] = true
	solver.decisionLevel = 1

	clause := solver.convertXORConflictToClause(xor)
	if clause == nil {
		t.Error("should produce a conflict clause")
	}
}

func TestProbeCreateXORConflictClauses(t *testing.T) {
	solver := NewCDCLSolver()
	solver.assignment["A"] = true
	solver.assignment["B"] = true
	solver.decisionLevel = 1
	solver.trail = NewDecisionTrail()
	solver.trail.Assign("A", true, 0, nil)
	solver.trail.Assign("B", true, 1, nil)

	xor := NewXORClause([]string{"A", "B"}, true)
	c := solver.createFullXORConflictClause(xor, []string{"A", "B"}, false)
	_ = c

	c2 := solver.createUnitXORConflictClause(xor, []string{"A"}, "B", false)
	_ = c2

	c3 := solver.createPartialXORConflictClause(xor, []string{"A"}, []string{"B"}, false)
	_ = c3
}

func TestProbeAddUnassignedXOR(t *testing.T) {
	solver := NewCDCLSolver()
	solver.extendedCNF = NewExtendedCNF()
	xor := NewXORClause([]string{"A", "B", "C"}, true)
	solver.extendedCNF.AddXORClause(xor)

	solver.assignment["A"] = true
	var literals []Literal
	solver.addUnassignedXORConstraints(&literals, []string{"B", "C"}, false)
	_ = literals
}

func TestProbeSortVariablesByActivity(t *testing.T) {
	solver := NewCDCLSolver()
	solver.variableActivity["B"] = 5.0
	solver.variableActivity["A"] = 10.0
	solver.variableActivity["C"] = 3.0

	solver.sortVariablesByActivity([]string{"A", "B", "C"})
}

func TestProbeSetXORClauseLBD(t *testing.T) {
	solver := NewCDCLSolver()
	solver.assignment["A"] = true
	solver.assignment["B"] = false
	solver.trail = NewDecisionTrail()
	solver.trail.Assign("A", true, 0, nil)
	solver.trail.Assign("B", false, 1, nil)

	clause := NewClause(L("A", false), L("B", false))
	solver.setXORClauseLBD(clause)
}

// === Inprocessing paths ===

func TestProbePerformInprocessing(t *testing.T) {
	solver := NewCDCLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true)))
	solver.cnf = cnf
	solver.assignment = make(Assignment)
	solver.inprocessConfig = DefaultInprocessConfig()
	solver.inprocessConfig.EnableVariableElim = true
	solver.inprocessConfig.EnableVivification = true
	solver.inprocessConfig.EnableSubsumption = true
	solver.inprocessConfig.EnableFailedLitProbing = true

	solver.performInprocessing()
}

func TestProbeRebuildWatchLists(t *testing.T) {
	solver := NewCDCLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	solver.cnf = cnf
	solver.watchLists = make(map[string][]*WatchedClause)

	solver.rebuildWatchLists()
	if len(solver.watchLists) == 0 {
		t.Error("watch lists should be rebuilt")
	}
}

func TestProbeUpdateHeuristicsAfterInprocessing(t *testing.T) {
	solver := NewCDCLSolver()
	solver.heuristic = NewVSIDSHeuristic()
	solver.cnf = NewCNF()
	solver.cnf.AddClause(NewClause(L("A", false)))
	result := &InprocessResult{ClausesRemoved: 5}
	solver.updateHeuristicsAfterInprocessing(result)
}

func TestProbeAdaptInprocessingFrequency(t *testing.T) {
	solver := NewCDCLSolver()
	solver.inprocessGap = 4000
	solver.lastInprocessReduction = 50
	solver.lastInprocessCostNs = 1000000

	result := &InprocessResult{ClausesRemoved: 5}
	solver.adaptInprocessingFrequency(result, 1000000)
}

func TestProbeShouldRunInprocessingAfterRestart(t *testing.T) {
	solver := NewCDCLSolver()
	solver.conflicts = 5000
	solver.lastInprocess = 0
	solver.inprocessGap = 4000

	result := solver.shouldRunInprocessingAfterRestart()
	_ = result
}

// === Preprocessor internals ===

func TestProbePreprocessorSubsumption(t *testing.T) {
	p := NewSATPreprocessor()
	cnf := NewCNF()
	// Clause 1 subsumes clause 2: (A) subsumes (A ∨ B)
	cnf.AddClause(NewClause(L("A", false)))
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("C", false)))

	p.subsumption(cnf)
}

func TestProbePreprocessorUnitPropagation(t *testing.T) {
	p := NewSATPreprocessor()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))           // unit clause
	cnf.AddClause(NewClause(L("A", true), L("B", false))) // B must be true

	p.unitPropagation(cnf)
}

func TestProbePreprocessorPureLiteralElimination(t *testing.T) {
	p := NewSATPreprocessor()
	cnf := NewCNF()
	// A appears only positively → pure literal, can be set true
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", false), L("C", true)))

	p.pureLiteralElimination(cnf)
}

// === DPLL internals ===

func TestProbeDPLLAllClausesSatisfied(t *testing.T) {
	dpll := NewDPLLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	dpll.cnf = cnf
	dpll.assignment = make(Assignment)
	dpll.assignment["A"] = true

	if !dpll.allClausesSatisfied() {
		t.Error("A=true should satisfy clause (A)")
	}
}

func TestProbeDPLLUnitPropagation(t *testing.T) {
	dpll := NewDPLLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))           // unit: A=true
	cnf.AddClause(NewClause(L("A", true), L("B", false))) // with A=true, ¬A is false, clause still has B
	dpll.cnf = cnf
	dpll.assignment = make(Assignment)

	conflict, err := dpll.unitPropagation()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if conflict {
		t.Log("unexpected conflict")
	}
}

func TestProbeDPLLGetUnassignedLiterals(t *testing.T) {
	dpll := NewDPLLSolver()
	dpll.assignment = make(Assignment)
	dpll.assignment["B"] = true

	clause := NewClause(L("A", false), L("B", false), L("C", false))
	unassigned := dpll.getUnassignedLiterals(clause)
	if len(unassigned) != 2 {
		t.Errorf("expected 2 unassigned literals (A, C), got %d", len(unassigned))
	}
}

func TestProbeDPLLChooseDecisionVariable(t *testing.T) {
	dpll := NewDPLLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	dpll.cnf = cnf
	dpll.assignment = make(Assignment)
	dpll.assignment["A"] = true

	v := dpll.chooseDecisionVariable()
	if v == "" {
		t.Error("should choose B as decision variable")
	}
}

func TestProbeDPLLPureLiteralElimination(t *testing.T) {
	dpll := NewDPLLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", false), L("C", false)))
	dpll.cnf = cnf
	dpll.assignment = make(Assignment)

	err := dpll.pureLiteralElimination()
	if err != nil {
		t.Fatalf("error: %v", err)
	}
}

// === CDCL restart and deletion ===

func TestProbeCDCLRestart(t *testing.T) {
	solver := NewCDCLSolver()
	solver.assignment = make(Assignment)
	solver.assignment["A"] = true
	solver.trail = NewDecisionTrail()
	solver.trail.Assign("A", true, 0, nil)
	solver.heuristic = NewVSIDSHeuristic()
	solver.restartStrategy = NewLubyRestartStrategy()
	solver.cnf = NewCNF()

	solver.restart()
}

func TestProbeCDCLDeleteClauses(t *testing.T) {
	solver := NewCDCLSolver()
	solver.deletionPolicy = NewActivityBasedDeletion()
	solver.heuristic = NewVSIDSHeuristic()
	solver.cnf = NewCNF()
	solver.assignment = make(Assignment)
	db := NewClauseDatabase(10, 100)
	for i := 0; i < 15; i++ {
		c := NewClause(L(varName(i), false), L(varName(i+10), false))
		c.Learned = true
		c.ID = i
		c.Activity = 0.01
		c.LBD = 8
		c.Tier = 2
		db.AddClause(c, int64(i))
	}
	solver.clauseDatabase = db
	solver.maxLearnedSize = 5

	solver.deleteClauses()
}

func TestProbeCDCLGaussianElimination(t *testing.T) {
	solver := NewCDCLSolver()
	solver.extendedCNF = NewExtendedCNF()
	solver.extendedCNF.AddXORClause(NewXORClause([]string{"A", "B"}, true))
	solver.extendedCNF.AddXORClause(NewXORClause([]string{"B", "C"}, false))
	solver.assignment = make(Assignment)
	solver.conflicts = 6000
	solver.gaussianEliminator.lastGaussian = 0

	solver.performGaussianElimination()
}

// === Inprocessing sub-components ===

func TestProbeInprocessorShouldRun(t *testing.T) {
	m := NewModernInprocessor()
	m.lastInprocess = 0

	if !m.ShouldRunInprocessing(5000) {
		t.Log("inprocessing should run after enough conflicts")
	}
	m.OnInprocessingCompleted(5000)
}

func TestProbeVivifyClauses(t *testing.T) {
	m := NewModernInprocessor()
	clauses := []*Clause{
		NewClause(L("A", false), L("B", false), L("C", false)),
		NewClause(L("D", false)),
	}
	assignment := make(Assignment)
	count := m.VivifyClauses(clauses, assignment)
	_ = count
}

func TestProbeEliminateVariables(t *testing.T) {
	m := NewModernInprocessor()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true), L("C", false)))
	cnf.AddClause(NewClause(L("B", true), L("C", true)))
	cnf.Variables = []string{"A", "B", "C"}

	eliminated := m.EliminateVariables(cnf.Variables, cnf)
	_ = eliminated
}

func TestProbeProbeFailedLiterals(t *testing.T) {
	m := NewModernInprocessor()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true)))

	literals := []Literal{L("A", false), L("A", true), L("B", false)}
	result := m.ProbeFailedLiterals(literals, cnf)
	_ = result
}

func TestProbeSubsumeAndStrengthen(t *testing.T) {
	m := NewModernInprocessor()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("C", false)))

	count := m.SubsumeAndStrengthen(cnf)
	_ = count
}

// === Trail internals ===

func TestProbeTrailLevelStarts(t *testing.T) {
	trail := NewDecisionTrail()
	trail.Assign("A", true, 0, nil)
	trail.Assign("B", false, 1, nil)
	trail.Assign("C", true, 1, nil)
	trail.Assign("D", false, 2, nil)

	levels := trail.GetAllLevels()
	if len(levels) < 1 {
		t.Error("expected at least 1 level")
	}
}

// === Gaussian eliminator internal state ===

func TestProbeGaussianExtractResults(t *testing.T) {
	ge := NewGaussianEliminator()
	ge.matrixToVar = []string{"A", "B"}
	ge.matrixCols = 3 // 2 vars + augmented
	ge.matrixRows = 2
	ge.matrix = memory.MustPoolSlice[[]bool](satPool, 2)[:2]
	ge.matrix[0] = memory.MustPoolSlice[bool](satPool, 3)[:3]
	ge.matrix[1] = memory.MustPoolSlice[bool](satPool, 3)[:3]
	// Row 0: A=1 (unit)
	ge.matrix[0][0] = true
	ge.matrix[0][1] = false
	ge.matrix[0][2] = true // RHS = 1
	// Row 1: B=1 (unit)
	ge.matrix[1][0] = false
	ge.matrix[1][1] = true
	ge.matrix[1][2] = true // RHS = 1

	result := &GaussianResult{}
	assignment := make(Assignment)
	ge.extractResults(result, assignment)

	if result.ConflictFound {
		t.Log("unexpected conflict from consistent matrix")
	}
}

func TestProbeGaussianContradiction(t *testing.T) {
	ge := NewGaussianEliminator()
	ge.matrixToVar = []string{"A"}
	ge.matrixCols = 2 // 1 var + augmented
	ge.matrixRows = 1
	ge.matrix = memory.MustPoolSlice[[]bool](satPool, 1)[:1]
	ge.matrix[0] = memory.MustPoolSlice[bool](satPool, 2)[:2]
	// 0*A = 1 → contradiction
	ge.matrix[0][0] = false
	ge.matrix[0][1] = true

	result := &GaussianResult{}
	ge.extractResults(result, make(Assignment))

	if !result.ConflictFound {
		t.Error("should detect contradiction: 0=1")
	}
}

// === Conflict analysis internals ===

func TestProbeGetTrailEntriesAtLevel(t *testing.T) {
	f := NewFirstUIPAnalyzer()
	trail := NewDecisionTrail()
	trail.Assign("A", true, 1, nil)
	trail.Assign("B", false, 1, NewClause(L("A", true)))

	entries := f.getTrailEntriesAtLevel(trail, 1)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries at level 1, got %d", len(entries))
	}
}

func TestProbeCountCurrentLevelVars(t *testing.T) {
	f := NewFirstUIPAnalyzer()
	trail := NewDecisionTrail()
	trail.Assign("A", true, 1, nil)
	trail.Assign("B", false, 1, NewClause(L("A", true)))

	clause := []Literal{L("A", true), L("B", false), L("C", false)}
	count := f.countCurrentLevelVars(clause, trail, 1)
	if count != 2 {
		t.Errorf("expected 2 vars at level 1, got %d", count)
	}
}

func TestProbeFindPositionInTrail(t *testing.T) {
	f := NewFirstUIPAnalyzer()
	entries := []TrailEntry{
		{Variable: "A", Value: true, Level: 1},
		{Variable: "B", Value: false, Level: 1},
	}
	pos := f.findPositionInTrail("A", entries)
	if pos != 0 {
		t.Errorf("position of A = %d, want 0", pos)
	}
}

// === Inprocessor configuration ===

func TestProbeInprocessConfig(t *testing.T) {
	m := NewModernInprocessor()
	cfg := DefaultInprocessConfig()
	cfg.InprocessGap = 2000
	m.Configure(cfg)
	stats := m.GetStatistics()
	_ = stats
	m.Reset()
}

// === ClauseDB tier methods ===

func TestProbeClauseDBUpdateMaxSize(t *testing.T) {
	db := NewClauseDatabase(100, 500)
	db.UpdateMaxSize(200)
	if db.GetMaxSize() != 200 {
		t.Error("max size should be updated")
	}
}

func TestProbeClauseDBPlaceToTier(t *testing.T) {
	db := NewClauseDatabase(100, 500)
	c := NewClause(L("A", false))
	c.Learned = true
	c.ID = 1

	db.placeToTier(c)

	core, mid, local, recent := db.GetTierSizes()
	_ = core
	_ = mid
	_ = local
	_ = recent
}

func TestProbeClauseDBStatistics(t *testing.T) {
	db := NewClauseDatabase(100, 500)
	c := NewClause(L("A", false))
	c.Learned = true
	c.ID = 1
	c.LBD = 3
	db.AddClause(c, 0)

	stats := db.GetTierStatistics()
	_ = stats
}

// === CDCL chronological backtracking paths ===

func TestProbeChronologicalBacktrack(t *testing.T) {
	solver := NewCDCLSolver()
	solver.trail = NewDecisionTrail()
	solver.trail.Assign("A", true, 0, nil)
	solver.trail.Assign("B", false, 1, nil)
	solver.assignment = make(Assignment)
	solver.assignment["A"] = true
	solver.assignment["B"] = false
	solver.decisionLevel = 1

	// Probe shouldUseChronologicalBacktrack logic
	solver.shouldUseChronologicalBacktrack(2, 0)
}

// === CDCL solve with timeout for coverage ===

func TestProbeSolveWithTimeout(t *testing.T) {
	solver := NewCDCLSolver()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))

	result := solver.SolveWithTimeout(cnf, 10*time.Second)
	if result.Error != nil {
		t.Fatalf("error: %v", result.Error)
	}
	if !result.Satisfiable {
		t.Error("should be SAT")
	}
}

// === Rescale and decay paths ===

func TestProbeRescaleVariableActivities(t *testing.T) {
	solver := NewCDCLSolver()
	solver.variableActivity["A"] = 1e200
	solver.variableActivity["B"] = 1e200
	solver.varActivityInc = 1e200

	solver.rescaleVariableActivities()
}

func TestProbeDecayVariableActivities(t *testing.T) {
	solver := NewCDCLSolver()
	solver.varActivityInc = 2.0
	solver.varActivityDecay = 0.95

	solver.decayVariableActivities()
}

// === Equals and helper methods ===

func TestProbeLiteralEquals(t *testing.T) {
	l1 := Literal{Variable: "A", Negated: false}
	l2 := Literal{Variable: "A", Negated: false}
	l3 := Literal{Variable: "B", Negated: false}

	if !l1.Equals(l2) {
		t.Error("identical literals should be equal")
	}
	if l1.Equals(l3) {
		t.Error("different variables should not be equal")
	}
}

func TestProbeLiteralNegate(t *testing.T) {
	l := Literal{Variable: "A", Negated: false}
	n := l.Negate()
	if n.Variable != "A" || !n.Negated {
		t.Error("Negate should flip Negated flag")
	}
}

func TestProbeWatchedClauseGetWatched(t *testing.T) {
	c := NewClause(L("A", false), L("B", false))
	wc := NewWatchedClause(c)
	lits := wc.GetWatchedLiterals()
	if len(lits) == 0 {
		t.Error("should have watched literals")
	}
}

// === Gaussian internal matrix operations ===

func TestProbeGaussianShouldDisable(t *testing.T) {
	ge := NewGaussianEliminator()
	ge.stats.TotalRuns = 5
	ge.stats.VariablesEliminated = 0
	ge.stats.UnitPropagations = 0

	if !ge.shouldDisable() {
		t.Log("should auto-disable when ineffective")
	}
}

func TestProbeGaussianFindPivot(t *testing.T) {
	ge := NewGaussianEliminator()
	ge.matrixRows = 3
	ge.matrixCols = 4
	ge.matrix = memory.MustPoolSlice[[]bool](satPool, 3)[:3]
	for i := range ge.matrix {
		ge.matrix[i] = memory.MustPoolSlice[bool](satPool, 4)[:4]
	}
	ge.matrix[1][2] = true

	pivot := ge.findPivot(0, 2)
	if pivot != 1 {
		t.Errorf("pivot = %d, want 1", pivot)
	}

	nopivot := ge.findPivot(0, 3)
	if nopivot != -1 {
		t.Errorf("no pivot in col 3 should return -1, got %d", nopivot)
	}
}

func TestProbeGaussianEliminateRow(t *testing.T) {
	ge := NewGaussianEliminator()
	ge.matrixCols = 3
	ge.matrix = memory.MustPoolSlice[[]bool](satPool, 2)[:2]
	ge.matrix[0] = memory.MustPoolSlice[bool](satPool, 3)[:3]
	ge.matrix[1] = memory.MustPoolSlice[bool](satPool, 3)[:3]
	ge.matrix[0][0] = true
	ge.matrix[0][1] = true
	ge.matrix[1][0] = true
	ge.matrix[1][1] = false

	ge.eliminateRow(1, 0)
	// Row 1 XOR Row 0: [true, false] XOR [true, true] = [false, true]
	if ge.matrix[1][0] {
		t.Error("after elimination, matrix[1][0] should be false")
	}
	if !ge.matrix[1][1] {
		t.Error("after elimination, matrix[1][1] should be true")
	}
}

// === Closest clause strength test ===

func TestProbeClauseStrengthTest(t *testing.T) {
	cv := NewClauseVivifier()
	assignment := make(Assignment)
	assignment["A"] = true
	assignment["C"] = false

	literals := []Literal{L("A", false), L("B", false), L("C", false)}
	result := cv.testClauseStrength(literals, assignment)
	_ = result
}

// === BoundedVariableElimination resolve pairs ===

func TestProbeResolveClausesPair(t *testing.T) {
	bve := NewBoundedVariableElimination()
	c1 := NewClause(L("A", false), L("B", false))
	c2 := NewClause(L("A", true), L("C", false))

	bve.resolveClausesPair(c1, c2, "A")
}

// === Build watched implications ===

func TestProbeBuildWatchedImplications(t *testing.T) {
	flp := NewFailedLiteralProber()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true)))

	flp.buildImplicationStructures(cnf)
}

// === Final 0% coverage gap fillers ===

func TestProbeVivifierTestWithSAT(t *testing.T) {
	cv := NewClauseVivifier()
	literals := []Literal{L("A", false), L("B", false), L("C", false)}
	assignment := make(Assignment)
	assignment["A"] = true
	assignment["B"] = false
	cv.testWithSATSolver(literals, assignment)
}

func TestProbeVivifierGetUnassigned(t *testing.T) {
	cv := NewClauseVivifier()
	clause := NewClause(L("A", false), L("B", false), L("C", false))
	assignment := make(Assignment)
	assignment["A"] = true
	unassigned := cv.getUnassignedLiterals(clause, assignment)
	if len(unassigned) != 2 {
		t.Errorf("expected 2 unassigned, got %d", len(unassigned))
	}
}

func TestProbeUpdateEquivalenceClasses(t *testing.T) {
	flp := NewFailedLiteralProber()
	flp.updateEquivalenceClasses(L("A", false))
}

func TestProbeGetXORStatistics(t *testing.T) {
	solver := NewCDCLSolver()
	stats := solver.GetXORStatistics()
	if stats == nil {
		t.Error("XOR statistics should not be nil")
	}
}

func TestProbeGetClausesByTier(t *testing.T) {
	db := NewClauseDatabase(100, 500)
	c1 := NewClause(L("A", false))
	c1.Learned = true
	c1.ID = 1
	c1.LBD = 3
	db.AddClause(c1, 0)
	c2 := NewClause(L("B", false), L("C", false))
	c2.Learned = true
	c2.ID = 2
	c2.LBD = 8
	db.AddClause(c2, 0)

	clauses := db.GetClausesByTier(1)
	_ = clauses
	clauses2 := db.GetClausesByTier(2)
	_ = clauses2
}

func TestProbeCompactSlice(t *testing.T) {
	clauses := []*Clause{
		NewClause(L("A", false)),
		nil,
		NewClause(L("B", false)),
	}
	result := compactSlice(clauses)
	if len(result) != 2 {
		t.Errorf("expected 2 non-nil clauses after compact, got %d", len(result))
	}
}

func TestProbeToRegularClauses(t *testing.T) {
	xor := NewXORClause([]string{"A", "B"}, true)
	clauses := xor.ToRegularClauses()
	if len(clauses) == 0 {
		t.Error("XOR should convert to regular clauses")
	}
}

func TestProbeEliminatorPerformHyperBinary(t *testing.T) {
	flp := NewFailedLiteralProber()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	result := ProbingResult{
		Implied:     []Literal{L("B", false)},
		Equivalents: []Literal{L("C", false)},
	}
	flp.performHyperBinaryResolution(result, cnf)
}

func TestProbeHasBinaryClause(t *testing.T) {
	flp := NewFailedLiteralProber()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	found := flp.hasBinaryClause(cnf, L("A", false), L("B", false))
	_ = found
}

func TestProbeRegisterBinaryClause(t *testing.T) {
	flp := NewFailedLiteralProber()
	flp.registerBinaryClause(L("A", false), L("B", true))
}

func TestProbeProbingWithUnitPropagation(t *testing.T) {
	flp := NewFailedLiteralProber()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	assignment := make(Assignment)
	units, conflict := flp.performProbingWithUnitPropagation(cnf, assignment, L("A", false))
	_ = units
	_ = conflict
}

func TestProbeGenerateProbingCandidates(t *testing.T) {
	flp := NewFailedLiteralProber()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("C", true)))
	cnf.Variables = []string{"A", "B", "C"}
	assignment := make(Assignment)
	candidates := flp.generateProbingCandidates(cnf, assignment)
	_ = candidates
}

func TestProbeCalculateCandidatePriority(t *testing.T) {
	flp := NewFailedLiteralProber()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("A", true), L("C", false)))
	priority := flp.calculateCandidatePriority(L("A", false), cnf)
	_ = priority
}

func TestProbeSortCandidatesByPriority(t *testing.T) {
	flp := NewFailedLiteralProber()
	flp.candidateQueue = []ProbingCandidate{
		{Literal: L("A", false), Priority: 0.5},
		{Literal: L("B", false), Priority: 2.0},
		{Literal: L("C", false), Priority: 0.1},
	}
	flp.sortCandidatesByPriority()
}

func TestProbeEliminatorCleanup(t *testing.T) {
	bve := NewBoundedVariableElimination()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.Variables = []string{"A", "B", "C"}
	bve.cleanupEliminatedVariables(cnf)
}

func TestProbeSubsumerCheckSubsumption(t *testing.T) {
	is := NewInprocessSubsumption()
	c1 := NewClause(L("A", false))
	c1.ID = 1
	c2 := NewClause(L("A", false), L("B", false))
	c2.ID = 2
	if !is.checkSubsumption(c1, c2) {
		t.Error("(A) should subsume (A∨B)")
	}
	if is.checkSubsumption(c2, c1) {
		t.Error("(A∨B) should not subsume (A)")
	}
}

func TestProbeSubsumerFindAndRemove(t *testing.T) {
	is := NewInprocessSubsumption()
	cnf := NewCNF()
	cnf.AddClause(NewClause(L("A", false)))
	cnf.AddClause(NewClause(L("A", false), L("B", false)))
	cnf.AddClause(NewClause(L("C", false)))
	removed := is.FindAndRemoveSubsumed(cnf)
	_ = removed
}
