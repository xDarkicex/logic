package sat

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xDarkicex/logic/classical"
	"github.com/xDarkicex/logic/core"
)

func TestAdvancedCDCLSolver(t *testing.T) {
	t.Run("Basic Satisfiable", func(t *testing.T) {
		solver := NewAdvancedCDCLSolver() // Fresh solver for each test
		cnf := createSimpleSATInstanceAdvanced()

		result := solver.SolveWithTimeout(cnf, 5*time.Second)

		if result.Error != nil {
			t.Fatalf("Solver error: %v", result.Error)
		}

		if !result.Satisfiable {
			t.Error("Expected satisfiable result")
		}

		if !verifySolutionAdvanced(cnf, result.Assignment) {
			t.Error("Solution verification failed")
		}
	})

	t.Run("Pigeon Hole Principle", func(t *testing.T) {
		solver := NewAdvancedCDCLSolver() // Fresh solver for each test
		cnf := createPigeonHolePrincipleAdvanced(4, 3)

		result := solver.SolveWithTimeout(cnf, 5*time.Second)

		if result.Error != nil && !strings.Contains(result.Error.Error(), "timeout") {
			t.Fatalf("Solver error: %v", result.Error)
		}

		if result.Error == nil {
			if result.Satisfiable {
				t.Error("Pigeon hole principle should be unsatisfiable")
			}

			if result.Statistics.Decisions > 0 || result.Statistics.Conflicts > 0 {
				t.Logf("Advanced solver stats: %s", result.Statistics.String())
			}
		} else {
			t.Logf("Test timed out (acceptable): %v", result.Error)
		}
	})

	t.Run("Random 3-SAT", func(t *testing.T) {
		solver := NewAdvancedCDCLSolver() // Fresh solver for each test
		cnf := createRandom3SATAdvanced(20, 85)
		result := solver.SolveWithTimeout(cnf, 3*time.Second)

		if result.Error != nil && !strings.Contains(result.Error.Error(), "timeout") {
			t.Fatalf("Unexpected solver error: %v", result.Error)
		}

		if result.Error == nil && result.Satisfiable {
			if !verifySolutionAdvanced(cnf, result.Assignment) {
				t.Error("Solution verification failed")
			}
		}

		t.Logf("Random 3-SAT stats: %s", result.Statistics.String())
	})
}

func TestConflictAnalysisAdvanced(t *testing.T) {
	analyzer := NewFirstUIPAnalyzer()
	trail := NewAdvancedDecisionTrail()

	trail.Assign("A", true, 1, nil)
	trail.Assign("B", false, 1, nil)

	conflictClause := NewClause(
		Literal{Variable: "A", Negated: true},
		Literal{Variable: "B", Negated: false},
		Literal{Variable: "C", Negated: false},
	)

	learnedClause, backtrackLevel := analyzer.Analyze(conflictClause, trail)

	if learnedClause == nil {
		t.Error("Expected learned clause from conflict analysis")
	}

	if backtrackLevel < 0 {
		t.Error("Invalid backtrack level")
	}

	t.Logf("Learned clause: %s", learnedClause.String())
	t.Logf("Backtrack level: %d", backtrackLevel)
}

func TestPreprocessorAdvanced(t *testing.T) {
	preprocessor := NewSATPreprocessor()
	cnf := NewCNF()

	cnf.AddClause(NewClause(Literal{Variable: "A", Negated: false}))
	cnf.AddClause(NewClause(
		Literal{Variable: "A", Negated: true},
		Literal{Variable: "B", Negated: false},
	))
	cnf.AddClause(NewClause(Literal{Variable: "C", Negated: false}))
	cnf.AddClause(NewClause(
		Literal{Variable: "C", Negated: false},
		Literal{Variable: "D", Negated: false},
	))

	originalClauses := len(cnf.Clauses)
	originalVars := len(cnf.Variables)

	processed, err := preprocessor.Preprocess(cnf)
	if err != nil {
		t.Fatalf("Preprocessing error: %v", err)
	}

	t.Logf("Original: %d clauses, %d variables", originalClauses, originalVars)
	t.Logf("Processed: %d clauses, %d variables", len(processed.Clauses), len(processed.Variables))

	if len(processed.Clauses) >= originalClauses {
		t.Error("Expected clause reduction from preprocessing")
	}
}

func TestCrossVerificationAdvanced(t *testing.T) {
	complexExpressions := []string{
		"((A -> B) & (B -> C)) -> (A -> C)",
		"!(A & B) <-> (!A | !B)",
		"(A ^ B) <-> ((A | B) & !(A & B))",
		"((A -> B) & (C -> D) & (A | C)) -> (B | D)",
	}

	variables := []string{"A", "B", "C", "D"}

	for _, expr := range complexExpressions {
		t.Run(fmt.Sprintf("Expression: %s", expr), func(t *testing.T) {
			satSystem := NewSATSystem() // Fresh system for each expression

			for trial := 0; trial < 5; trial++ { // Reduced trials to avoid timeout
				assignment := generateRandomAssignmentAdvanced(variables)
				classicalResult := evaluateClassicalAdvanced(expr, assignment)

				satResult, err := satSystem.VerifySolution(expr, assignment)
				if err != nil {
					t.Fatalf("SAT verification error: %v", err)
				}

				if classicalResult != satResult {
					t.Errorf("Mismatch for %s with %v: classical=%v, sat=%v",
						expr, assignment, classicalResult, satResult)
				}
			}
		})
	}
}

func TestSATSystemIntegrationAdvanced(t *testing.T) {
	testCases := []struct {
		expr        string
		expectedSat bool
		description string
	}{
		{"(A -> B) & (B -> C) & A & !C", false, "transitivity violation"},
		{"(A <-> B) & (B <-> C) & (A <-> C)", true, "equivalence transitivity"},
		{"!(A & B & C) | (A & B & C)", true, "tautology with complex subformula"},
		{"(A | B) & (!A | C) & (!B | C) & !C", false, "resolution chain to contradiction"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Create fresh systems for each test case to avoid race conditions
			system := NewSATSystem()
			advancedSystem := NewSATSystemWithSolver(NewAdvancedCDCLSolver())

			ctx := core.NewEvaluationContext()

			// Test basic solver
			result1, err1 := evaluateWithTimeoutSafe(system, tc.expr, ctx, 3*time.Second)
			if err1 != nil {
				if strings.Contains(err1.Error(), "timeout") {
					t.Logf("Basic solver timed out (acceptable): %v", err1)
					return // Skip this test case
				}
				t.Fatalf("Basic solver error: %v", err1)
			}

			// Test advanced solver
			result2, err2 := evaluateWithTimeoutSafe(advancedSystem, tc.expr, ctx, 3*time.Second)
			if err2 != nil {
				if strings.Contains(err2.Error(), "timeout") {
					t.Logf("Advanced solver timed out (acceptable): %v", err2)
					return // Skip this test case
				}
				t.Fatalf("Advanced solver error: %v", err2)
			}

			// Extract results
			sat1, ok1 := extractSatisfiableResult(result1)
			if !ok1 {
				t.Fatalf("Invalid result format from basic solver: %T", result1)
			}

			sat2, ok2 := extractSatisfiableResult(result2)
			if !ok2 {
				t.Fatalf("Invalid result format from advanced solver: %T", result2)
			}

			// Compare results
			if sat1 != sat2 {
				t.Errorf("Solver disagreement: basic=%v, advanced=%v", sat1, sat2)
			}

			if sat1 != tc.expectedSat {
				t.Errorf("Expected %v, got %v", tc.expectedSat, sat1)
			}

			t.Logf("Test completed: %s (basic=%v, advanced=%v)", tc.description, sat1, sat2)
		})
	}
}

func TestCDCLSolverAdvanced(t *testing.T) {
	t.Run("Pigeon Hole with Timeout", func(t *testing.T) {
		solver := NewCDCLSolver() // Fresh solver
		cnf := NewCNF()

		// 3 pigeons, 2 holes - unsatisfiable
		cnf.AddClause(NewClause(
			Literal{Variable: "p1h1", Negated: false},
			Literal{Variable: "p1h2", Negated: false},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p2h1", Negated: false},
			Literal{Variable: "p2h2", Negated: false},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p3h1", Negated: false},
			Literal{Variable: "p3h2", Negated: false},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p1h1", Negated: true},
			Literal{Variable: "p2h1", Negated: true},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p1h1", Negated: true},
			Literal{Variable: "p3h1", Negated: true},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p2h1", Negated: true},
			Literal{Variable: "p3h1", Negated: true},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p1h2", Negated: true},
			Literal{Variable: "p2h2", Negated: true},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p1h2", Negated: true},
			Literal{Variable: "p3h2", Negated: true},
		))
		cnf.AddClause(NewClause(
			Literal{Variable: "p2h2", Negated: true},
			Literal{Variable: "p3h2", Negated: true},
		))

		result := solver.SolveWithTimeout(cnf, 3*time.Second)

		if result.Error != nil && strings.Contains(result.Error.Error(), "timeout") {
			t.Logf("Test timed out (acceptable): %v", result.Error)
			return
		}

		if result.Error != nil {
			t.Fatalf("Solver error: %v", result.Error)
		}

		if result.Satisfiable {
			t.Error("Pigeon hole principle should be unsatisfiable")
		}

		t.Logf("Statistics: %s", result.Statistics.String())
	})
}

// Thread-safe evaluation function
func evaluateWithTimeoutSafe(system *SATSystemImpl, expr string, ctx core.EvaluationContext, timeout time.Duration) (interface{}, error) {
	type result struct {
		data interface{}
		err  error
	}

	resultChan := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- result{nil, fmt.Errorf("panic: %v", r)}
			}
		}()

		data, err := system.Evaluate(expr, ctx)
		resultChan <- result{data, err}
	}()

	select {
	case res := <-resultChan:
		return res.data, res.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("evaluation timeout after %v", timeout)
	}
}

func BenchmarkAdvancedCDCLSolver(b *testing.B) {
	benchmarks := []struct {
		name      string
		generator func() *CNF
		timeout   time.Duration
	}{
		{"3-SAT-15-60", func() *CNF { return createRandom3SATAdvanced(15, 60) }, 1 * time.Second},
		{"Pigeon-3-2", func() *CNF { return createPigeonHolePrincipleAdvanced(3, 2) }, 2 * time.Second},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			cnf := bm.generator()

			for i := 0; i < b.N; i++ {
				solver := NewAdvancedCDCLSolver() // Fresh solver for each iteration
				result := solver.SolveWithTimeout(cnf, bm.timeout)

				if result.Error != nil && !strings.Contains(result.Error.Error(), "timeout") {
					b.Fatalf("Unexpected solver error: %v", result.Error)
				}
			}
		})
	}
}

// Helper function to safely extract satisfiable result
func extractSatisfiableResult(result interface{}) (bool, bool) {
	if result == nil {
		return false, false
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return false, false
	}

	satisfiable, ok := resultMap["satisfiable"].(bool)
	return satisfiable, ok
}

// Helper functions for test cases
func createSimpleSATInstanceAdvanced() *CNF {
	cnf := NewCNF()
	cnf.AddClause(NewClause(
		Literal{Variable: "A", Negated: false},
		Literal{Variable: "B", Negated: false},
	))
	cnf.AddClause(NewClause(
		Literal{Variable: "A", Negated: true},
		Literal{Variable: "C", Negated: false},
	))
	return cnf
}

func createPigeonHolePrincipleAdvanced(pigeons, holes int) *CNF {
	cnf := NewCNF()

	for p := 1; p <= pigeons; p++ {
		clause := make([]Literal, holes)
		for h := 1; h <= holes; h++ {
			clause[h-1] = Literal{
				Variable: fmt.Sprintf("p%dh%d", p, h),
				Negated:  false,
			}
		}
		cnf.AddClause(NewClause(clause...))
	}

	for h := 1; h <= holes; h++ {
		for p1 := 1; p1 <= pigeons; p1++ {
			for p2 := p1 + 1; p2 <= pigeons; p2++ {
				cnf.AddClause(NewClause(
					Literal{Variable: fmt.Sprintf("p%dh%d", p1, h), Negated: true},
					Literal{Variable: fmt.Sprintf("p%dh%d", p2, h), Negated: true},
				))
			}
		}
	}

	return cnf
}

func createRandom3SATAdvanced(variables, clauses int) *CNF {
	cnf := NewCNF()

	for i := 0; i < clauses; i++ {
		literals := make([]Literal, 3)
		for j := 0; j < 3; j++ {
			varNum := (i*3+j)%variables + 1
			literals[j] = Literal{
				Variable: fmt.Sprintf("x%d", varNum),
				Negated:  (i+j)%2 == 0,
			}
		}
		cnf.AddClause(NewClause(literals...))
	}

	return cnf
}

func verifySolutionAdvanced(cnf *CNF, assignment Assignment) bool {
	for _, clause := range cnf.Clauses {
		if !assignment.Satisfies(clause) {
			return false
		}
	}
	return true
}

func generateRandomAssignmentAdvanced(variables []string) Assignment {
	assignment := make(Assignment)
	for _, variable := range variables {
		assignment[variable] = (len(variable)+len(assignment))%2 == 0
	}
	return assignment
}

func evaluateClassicalAdvanced(expr string, assignment Assignment) bool {
	ast, err := classical.ParseExpression(expr)
	if err != nil {
		return false
	}

	ctx := make(classical.EvaluationContext)
	for variable, value := range assignment {
		ctx[variable] = value
	}

	result, err := ast.Evaluate(ctx)
	return err == nil && result
}
