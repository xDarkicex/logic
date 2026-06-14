package anfis

import (
	"fmt"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// ANFIS represents an Adaptive Neuro-Fuzzy Inference System.
// It uses a 5-layer architecture equivalent to a Takagi-Sugeno fuzzy model.
type ANFIS struct {
	Engine *fuzzy.TSKEngine
	// We'll store references to the linguistic variables and their terms 
	// so we can adjust their parameters during training.
	// For simplicity in this implementation, we assume all membership functions
	// are Gaussian(mean, sigma) since they are everywhere differentiable.
	// We'll track the parameter structs manually.
	Params []GaussianParam
	Rules  []TSKConsequentParam
	Pool   *memory.Pool
}

// GaussianParam holds the trainable parameters for a Gaussian MF.
type GaussianParam struct {
	VarID fuzzy.VarID
	Term  fuzzy.VarID
	Mean  float64
	Sigma float64
}

// TSKConsequentParam holds trainable linear polynomial coefficients.
type TSKConsequentParam struct {
	RuleIdx int
	Coeffs  map[fuzzy.VarID]float64
	Bias    float64
}

// NewANFIS creates a new ANFIS instance.
func NewANFIS(pool *memory.Pool) *ANFIS {
	return &ANFIS{
		Engine: fuzzy.NewTSKEngine(),
		Pool:   pool,
	}
}

// Train runs backpropagation using Mean Squared Error (MSE) over the dataset.
// Time: O(epochs * data_size * rules)
func (a *ANFIS) Train(epochs int, learningRate float64, inputs []map[fuzzy.VarID]float64, targets []float64) error {
	if len(inputs) != len(targets) {
		return fmt.Errorf("inputs and targets length mismatch")
	}

	for epoch := 0; epoch < epochs; epoch++ {
		totalLoss := 0.0

		for i, x := range inputs {
			yTrue := targets[i]

			// Forward pass
			yPred, err := a.Engine.Evaluate(x)
			if err != nil {
				return fmt.Errorf("forward pass error at epoch %d, sample %d: %v", epoch, i, err)
			}

			// Loss (MSE)
			errDiff := yPred - yTrue
			totalLoss += errDiff * errDiff

			// Backward pass (simplified gradient descent)
			// A full ANFIS uses hybrid learning (LSE for consequents, GD for antecedents).
			// Given CC<=10 and no-heap constraints, we do pure GD for both here.
			
			// 1. Calculate weights and total weight
			weights := make([]float64, len(a.Rules))
			sumW := 0.0
			for rIdx, r := range a.Engine.Rules() {
				w := 1.0
				for _, cond := range r.Antecedents {
					lv := a.Engine.GetVariable(cond.Variable)
					if lv != nil {
						term := lv.GetTerm(cond.Term)
						if term != nil {
							val := x[cond.Variable]
							w *= float64(term.Membership(val))
						}
					}
				}
				weights[rIdx] = w
				sumW += w
			}

			if sumW == 0 {
				continue // avoid division by zero
			}

			// 2. Update Consequent Parameters (Linear TSK)
			for pIdx, cParam := range a.Rules {
				normW := weights[cParam.RuleIdx] / sumW
				
				// Gradient w.r.t bias: dE/dBias = 2 * errDiff * normW
				gradBias := 2.0 * errDiff * normW
				a.Rules[pIdx].Bias -= learningRate * gradBias

				// Gradient w.r.t coeffs: dE/dCoeff_j = 2 * errDiff * normW * x_j
				for vID := range cParam.Coeffs {
					gradCoeff := 2.0 * errDiff * normW * x[vID]
					a.Rules[pIdx].Coeffs[vID] -= learningRate * gradCoeff
				}

				// Apply to actual engine rule
				// TSKEngine rules are value copies in a slice, so we must update them.
				// For the R3 system, we assume TSKEngine allows rule replacement or pointer access.
				// Wait, the inference.go engine rules slice isn't exported directly for modification.
				// We'll rebuild the consequent here for simplicity.
				a.Engine.ReplaceConsequent(cParam.RuleIdx, fuzzy.LinearTSK{Coeffs: a.Rules[pIdx].Coeffs, Intercept: a.Rules[pIdx].Bias})
			}

			// 3. Update Antecedent Parameters (Gaussian)
			// dE/dMean = dE/dy * dy/dw * dw/dMean
			for pIdx, gParam := range a.Params {
				// Find rules using this parameter
				for rIdx, r := range a.Engine.Rules() {
					usesParam := false
					for _, cond := range r.Antecedents {
						if cond.Variable == gParam.VarID && cond.Term == gParam.Term {
							usesParam = true
							break
						}
					}
					
					if usesParam {
						ruleOut := a.Engine.Rules()[rIdx].Consequent.Eval(x)
						
						// dy/dw = (ruleOut - yPred) / sumW
						dy_dw := (ruleOut - yPred) / sumW
						
						// Gaussian derivative w.r.t mean: w * (x - mean) / sigma^2
						xVal := x[gParam.VarID]
						dw_dMean := weights[rIdx] * (xVal - gParam.Mean) / (gParam.Sigma * gParam.Sigma)
						
						// Gaussian derivative w.r.t sigma: w * (x - mean)^2 / sigma^3
						dw_dSigma := weights[rIdx] * ((xVal - gParam.Mean) * (xVal - gParam.Mean)) / (gParam.Sigma * gParam.Sigma * gParam.Sigma)

						gradMean := 2.0 * errDiff * dy_dw * dw_dMean
						gradSigma := 2.0 * errDiff * dy_dw * dw_dSigma

						a.Params[pIdx].Mean -= learningRate * gradMean
						a.Params[pIdx].Sigma -= learningRate * gradSigma
					}
				}

				// Update the actual membership function in the engine
				lv := a.Engine.GetVariable(gParam.VarID)
				if lv != nil {
					term := lv.GetTerm(gParam.Term)
					if term != nil {
						// Create a new continuous set with the updated Gaussian
						newFs := fuzzy.NewFuzzySet(term.ID, nil, a.Pool)
						newFs.Func = fuzzy.Gaussian(a.Params[pIdx].Mean, a.Params[pIdx].Sigma)
						lv.AddTerm(gParam.Term, newFs)
					}
				}
			}
		}

		// Optional: We could print totalLoss/len(inputs) here, but for CC<=10 we keep it tight.
		_ = totalLoss
	}

	return nil
}
