package sat

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xDarkicex/logic/classical"
	"github.com/xDarkicex/logic/core"
)

func TestSATSystemIntegration(t *testing.T) {
	system := NewSATSystem()

	// Test system interface compliance
	if system.Name() != "sat" {
		t.Errorf("Expected system name 'sat', got %s", system.Name())
	}

	// Test simple expressions
	testCases := []struct {
		expr        string
		expectedSat bool
		description string
	}{
		{"A", true, "single variable"},
		{"A & !A", false, "contradiction"},
		{"A | !A", true, "tautology"},
		{"(A & B) | (!A & !B)", true, "satisfiable formula"},
		{"A & B & !A", false, "simple contradiction"},
		{"(A -> B) & A & !B", false, "modus ponens contradiction"},
		{"(A <-> B) & (A | B) & !(A & B)", false, "biconditional contradiction"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			ctx := core.NewEvaluationContext()
			result, err := system.Evaluate(tc.expr, ctx)

			if err != nil {
				t.Fatalf("Evaluation error: %v", err)
			}

			resultMap := result.(map[string]interface{})
			satisfiable := resultMap["satisfiable"].(bool)

			if satisfiable != tc.expectedSat {
				t.Errorf("Expression %s: expected %v, got %v",
					tc.expr, tc.expectedSat, satisfiable)
			}

			if satisfiable {
				// Verify solution
				assignment := resultMap["assignment"].(Assignment)
				verified, err := system.VerifySolution(tc.expr, assignment)
				if err != nil {
					t.Errorf("Verification error: %v", err)
				}
				if !verified {
					t.Errorf("Solution verification failed for %s", tc.expr)
				}
			}
		})
	}
}

func TestCDCLSolver(t *testing.T) {
	solver := NewCDCLSolver()

	// Test pigeon hole principle (unsatisfiable)
	cnf := NewCNF()

	// 3 pigeons, 2 holes - unsatisfiable
	// Variables: p1h1, p1h2, p2h1, p2h2, p3h1, p3h2

	// Each pigeon must be in at least one hole
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

	// No two pigeons in same hole
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

	result := solver.Solve(cnf)

	if result.Error != nil {
		t.Fatalf("Solver error: %v", result.Error)
	}

	if result.Satisfiable {
		t.Error("Pigeon hole principle should be unsatisfiable")
	}

	stats := result.Statistics
	if stats.Conflicts == 0 {
		t.Error("Expected conflicts in pigeon hole problem")
	}

	t.Logf("Statistics: %s", stats.String())
}

func TestMAXSATSolver(t *testing.T) {
	maxsat := NewMAXSATSolver()

	// Create test instance with conflicting clauses
	cnf := NewCNF()

	// Clauses: A, !A, B
	// Should satisfy 2 out of 3 clauses (either {A,B} or {!A,B})
	cnf.AddClause(NewClause(Literal{Variable: "A", Negated: false}))
	cnf.AddClause(NewClause(Literal{Variable: "A", Negated: true}))
	cnf.AddClause(NewClause(Literal{Variable: "B", Negated: false}))

	weights := []float64{1.0, 1.0, 1.0} // Equal weights

	result := maxsat.SolveMAXSAT(cnf, weights)

	if result.Error != nil {
		t.Fatalf("MAX-SAT solver error: %v", result.Error)
	}

	if result.SatisfiedCount != 2 {
		t.Errorf("Expected 2 satisfied clauses, got %d", result.SatisfiedCount)
	}

	if result.TotalWeight != 2.0 {
		t.Errorf("Expected total weight 2.0, got %f", result.TotalWeight)
	}
}

func TestSolverTimeout(t *testing.T) {
	solver := NewCDCLSolver()

	// Create large unsatisfiable instance
	cnf := NewCNF()

	// Add many contradictory clauses
	for i := 0; i < 100; i++ {
		varName := fmt.Sprintf("x%d", i)
		cnf.AddClause(NewClause(Literal{Variable: varName, Negated: false}))
		cnf.AddClause(NewClause(Literal{Variable: varName, Negated: true}))
	}

	// Short timeout
	result := solver.SolveWithTimeout(cnf, 1*time.Millisecond)

	if result.Error == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(result.Error.Error(), "timeout") {
		t.Errorf("Expected timeout error, got: %v", result.Error)
	}
}

func BenchmarkCDCLSolver(b *testing.B) {
	solver := NewCDCLSolver()

	// Create satisfiable 3-SAT instance
	cnf := NewCNF()

	// 50 variables, 200 clauses
	for i := 0; i < 200; i++ {
		literals := make([]Literal, 3)
		for j := 0; j < 3; j++ {
			varNum := (i*3 + j) % 50
			literals[j] = Literal{
				Variable: fmt.Sprintf("x%d", varNum),
				Negated:  (i+j)%2 == 0,
			}
		}
		cnf.AddClause(NewClause(literals...))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		solver.Reset()
		result := solver.Solve(cnf)
		if result.Error != nil {
			b.Fatalf("Solver error: %v", result.Error)
		}
	}
}

func TestCrossVerificationWithClassical(t *testing.T) {
	satSystem := NewSATSystem()

	// Test expressions that should match classical evaluation
	expressions := []string{
		"A & B",
		"A | B",
		"A -> B",
		"A <-> B",
		"(A & B) | (C & D)",
		"!(A & B) <-> (!A | !B)", // De Morgan's law
	}

	variables := []string{"A", "B", "C", "D"}

	for _, expr := range expressions {
		t.Run(expr, func(t *testing.T) {
			// Test all possible assignments
			for i := 0; i < (1 << len(variables)); i++ {
				assignment := make(Assignment)
				classicalCtx := make(classical.EvaluationContext)

				for j, variable := range variables {
					value := (i>>j)&1 == 1
					assignment[variable] = value
					classicalCtx[variable] = value
				}

				// Classical evaluation
				ast, err := classical.ParseExpression(expr)
				if err != nil {
					t.Fatalf("Parse error: %v", err)
				}

				classicalResult, err := ast.Evaluate(classicalCtx)
				if err != nil {
					t.Fatalf("Classical evaluation error: %v", err)
				}

				// SAT verification
				satVerified, err := satSystem.VerifySolution(expr, assignment)
				if err != nil {
					t.Fatalf("SAT verification error: %v", err)
				}

				if classicalResult != satVerified {
					t.Errorf("Mismatch for %s with assignment %v: classical=%v, sat=%v",
						expr, assignment, classicalResult, satVerified)
				}
			}
		})
	}
}
