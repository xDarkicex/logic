package sat

import (
	"math/rand"

	"github.com/xDarkicex/logic/fuzzy"
)

// FuzzyLiteral represents a variable in a clause.
type FuzzyLiteral struct {
	VarID   fuzzy.VarID
	Negated bool
}

// FuzzyClause is a disjunction (OR) of fuzzy literals.
type FuzzyClause struct {
	Literals []FuzzyLiteral
}

// SolveFuzzy applies continuous gradient descent (NDProp) to find a satisfying
// assignment in [0,1] for the given fuzzy constraints.
// It returns the assignment map and a boolean indicating if it fully satisfied (loss < epsilon).
// CC=6, Time: O(epochs * clauses * literals), Space: O(vars)
func SolveFuzzy(clauses []FuzzyClause, variables []fuzzy.VarID, epochs int, learningRate float64) (map[fuzzy.VarID]float64, bool) {
	// Initialize assignments randomly in [0,1]
	assignment := make(map[fuzzy.VarID]float64, len(variables))
	for _, v := range variables {
		assignment[v] = rand.Float64()
	}

	for epoch := 0; epoch < epochs; epoch++ {
		totalLoss := 0.0
		
		// To accumulate gradients
		grads := make(map[fuzzy.VarID]float64, len(variables))

		for _, clause := range clauses {
			// Probabilistic T-Conorm: a OR b = a + b - a*b
			// For simplicity and gradient stability in NDProp, we can use the continuous
			// Lukasiewicz T-conorm: min(1, sum(x_i)) or a smooth approximation like sum(x_i).
			// The NDProp paper uses probabilistic T-conorm, but for multiple literals
			// it's 1 - Product(1 - x_i).
			
			// Let's compute clause truth C = 1 - Product(1 - l_i)
			prod := 1.0
			for _, lit := range clause.Literals {
				val := assignment[lit.VarID]
				if lit.Negated {
					val = 1.0 - val
				}
				prod *= (1.0 - val)
			}
			// Loss = (1 - clauseTruth)^2 = prod^2
			totalLoss += prod * prod

			// Gradient w.r.t each variable in the clause
			// dL/dx = 2 * prod * d(prod)/dx
			for _, lit := range clause.Literals {
				val := assignment[lit.VarID]
				if lit.Negated {
					val = 1.0 - val
				}
				
				// Avoid division by zero
				var dProd_dVal float64
				if (1.0 - val) < 1e-6 {
					// Recompute without this literal
					p2 := 1.0
					for _, l2 := range clause.Literals {
						if l2.VarID != lit.VarID {
							v2 := assignment[l2.VarID]
							if l2.Negated {
								v2 = 1.0 - v2
							}
							p2 *= (1.0 - v2)
						}
					}
					dProd_dVal = -p2
				} else {
					dProd_dVal = -prod / (1.0 - val)
				}

				if lit.Negated {
					dProd_dVal = -dProd_dVal // Chain rule for (1 - x)
				}

				grad := 2.0 * prod * dProd_dVal
				grads[lit.VarID] += grad
			}
		}

		if totalLoss < 1e-4 {
			return assignment, true
		}

		// Apply gradients
		for _, v := range variables {
			newVal := assignment[v] - learningRate*grads[v]
			// Clamp to [0,1]
			if newVal > 1.0 {
				newVal = 1.0
			} else if newVal < 0.0 {
				newVal = 0.0
			}
			assignment[v] = newVal
		}
	}

	return assignment, false
}
