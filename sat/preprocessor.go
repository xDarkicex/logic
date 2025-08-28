package sat

// SATPreprocessor implements advanced CNF simplification techniques
type SATPreprocessor struct {
	originalVars   []string
	eliminatedVars map[string]bool
	substitutions  map[string]Literal
	eliminated     int
}

func NewSATPreprocessor() *SATPreprocessor {
	return &SATPreprocessor{
		eliminatedVars: make(map[string]bool),
		substitutions:  make(map[string]Literal),
	}
}

func (p *SATPreprocessor) Preprocess(cnf *CNF) (*CNF, error) {
	p.originalVars = make([]string, len(cnf.Variables))
	copy(p.originalVars, cnf.Variables)

	result := &CNF{
		Clauses:   make([]*Clause, 0, len(cnf.Clauses)),
		Variables: make([]string, 0, len(cnf.Variables)),
		nextID:    cnf.nextID,
	}

	// Copy clauses
	for _, clause := range cnf.Clauses {
		newClause := &Clause{
			Literals: make([]Literal, len(clause.Literals)),
			ID:       clause.ID,
			Learned:  clause.Learned,
			Activity: clause.Activity,
		}
		copy(newClause.Literals, clause.Literals)
		result.Clauses = append(result.Clauses, newClause)
	}

	// Apply preprocessing techniques
	changed := true
	rounds := 0

	for changed && rounds < 10 {
		changed = false
		rounds++

		// Unit propagation
		if p.unitPropagation(result) {
			changed = true
		}

		// Pure literal elimination
		if p.pureLiteralElimination(result) {
			changed = true
		}

		// Subsumption
		if p.subsumption(result) {
			changed = true
		}

		// Self-subsuming resolution
		if p.selfSubsumingResolution(result) {
			changed = true
		}
	}

	// Update variables list
	p.updateVariables(result)

	return result, nil
}

func (p *SATPreprocessor) unitPropagation(cnf *CNF) bool {
	changed := false
	iterations := 0

	for iterations < 10 { // Limit iterations
		iterations++
		unitClause := p.findUnitClause(cnf)
		if unitClause == nil {
			break
		}

		unit := unitClause.Literals[0]
		changed = true

		// Remove satisfied clauses and falsified literals
		newClauses := make([]*Clause, 0)

		for _, clause := range cnf.Clauses {
			if clause == unitClause {
				continue // Remove unit clause
			}

			// Check if clause is satisfied by unit
			satisfied := false
			for _, lit := range clause.Literals {
				if lit.Equals(unit) {
					satisfied = true
					break
				}
			}

			if satisfied {
				continue // Remove satisfied clause
			}

			// Remove negated unit literal
			newLiterals := make([]Literal, 0)
			for _, lit := range clause.Literals {
				if !lit.Equals(unit.Negate()) {
					newLiterals = append(newLiterals, lit)
				}
			}

			if len(newLiterals) == 0 {
				// Empty clause - contradiction
				cnf.Clauses = []*Clause{}
				return changed
			}

			newClause := &Clause{
				Literals: newLiterals,
				ID:       clause.ID,
				Learned:  clause.Learned,
				Activity: clause.Activity,
			}
			newClauses = append(newClauses, newClause)
		}

		cnf.Clauses = newClauses
		p.substitutions[unit.Variable] = unit
		p.eliminatedVars[unit.Variable] = true
		p.eliminated++
	}

	return changed
}

func (p *SATPreprocessor) pureLiteralElimination(cnf *CNF) bool {
	literalCount := make(map[string]int)

	// Count occurrences
	for _, clause := range cnf.Clauses {
		for _, lit := range clause.Literals {
			if !p.eliminatedVars[lit.Variable] {
				if lit.Negated {
					literalCount[lit.Variable]--
				} else {
					literalCount[lit.Variable]++
				}
			}
		}
	}

	// Find pure literals
	pureLiterals := make([]Literal, 0)
	for variable, count := range literalCount {
		if count > 0 {
			pureLiterals = append(pureLiterals, Literal{Variable: variable, Negated: false})
		} else if count < 0 {
			pureLiterals = append(pureLiterals, Literal{Variable: variable, Negated: true})
		}
	}

	if len(pureLiterals) == 0 {
		return false
	}

	// Remove clauses containing pure literals
	newClauses := make([]*Clause, 0, len(cnf.Clauses))

	for _, clause := range cnf.Clauses {
		satisfied := false
		for _, pureLit := range pureLiterals {
			if p.clauseContains(clause, pureLit) {
				satisfied = true
				break
			}
		}

		if !satisfied {
			newClauses = append(newClauses, clause)
		}
	}

	// Mark variables as eliminated
	for _, lit := range pureLiterals {
		p.substitutions[lit.Variable] = lit
		p.eliminatedVars[lit.Variable] = true
		p.eliminated++
	}

	cnf.Clauses = newClauses
	return true
}

func (p *SATPreprocessor) subsumption(cnf *CNF) bool {
	changed := false

	for i := 0; i < len(cnf.Clauses); i++ {
		for j := i + 1; j < len(cnf.Clauses); j++ {
			if p.subsumes(cnf.Clauses[i], cnf.Clauses[j]) {
				// Remove subsumed clause j
				cnf.Clauses = append(cnf.Clauses[:j], cnf.Clauses[j+1:]...)
				j--
				changed = true
			} else if p.subsumes(cnf.Clauses[j], cnf.Clauses[i]) {
				// Remove subsumed clause i
				cnf.Clauses = append(cnf.Clauses[:i], cnf.Clauses[i+1:]...)
				i--
				changed = true
				break
			}
		}
	}

	return changed
}

func (p *SATPreprocessor) selfSubsumingResolution(cnf *CNF) bool {
	// Implementation would go here
	return false
}

func (p *SATPreprocessor) PostProcess(assignment Assignment) Assignment {
	result := make(Assignment)

	// Copy non-eliminated variables
	for variable, value := range assignment {
		if !p.eliminatedVars[variable] {
			result[variable] = value
		}
	}

	// Add eliminated variables with their forced values
	for variable, literal := range p.substitutions {
		result[variable] = !literal.Negated
	}

	return result
}

// Helper methods
func (p *SATPreprocessor) findUnitClause(cnf *CNF) *Clause {
	for _, clause := range cnf.Clauses {
		if len(clause.Literals) == 1 && !p.eliminatedVars[clause.Literals[0].Variable] {
			return clause
		}
	}
	return nil
}

func (p *SATPreprocessor) clauseContains(clause *Clause, literal Literal) bool {
	for _, lit := range clause.Literals {
		if lit.Equals(literal) {
			return true
		}
	}
	return false
}

func (p *SATPreprocessor) subsumes(c1, c2 *Clause) bool {
	for _, lit1 := range c1.Literals {
		found := false
		for _, lit2 := range c2.Literals {
			if lit1.Equals(lit2) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (p *SATPreprocessor) updateVariables(cnf *CNF) {
	varSet := make(map[string]bool)

	for _, clause := range cnf.Clauses {
		for _, lit := range clause.Literals {
			if !p.eliminatedVars[lit.Variable] {
				varSet[lit.Variable] = true
			}
		}
	}

	cnf.Variables = make([]string, 0, len(varSet))
	for variable := range varSet {
		cnf.Variables = append(cnf.Variables, variable)
	}
}
