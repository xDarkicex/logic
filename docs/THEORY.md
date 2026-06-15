# SAT Solver Theory

## Propositional Satisfiability (SAT)

SAT is the canonical NP-complete problem: given a Boolean formula, determine whether
there exists an assignment of truth values to variables that makes the formula true.
Despite worst-case exponential complexity, modern SAT solvers routinely solve
industrial instances with millions of variables and clauses.

A formula is in **Conjunctive Normal Form (CNF)** when expressed as a conjunction
(AND) of clauses, where each clause is a disjunction (OR) of literals. A literal is
a variable or its negation. Example:

```
(A ∨ ¬B) ∧ (¬A ∨ C ∨ ¬D) ∧ (B ∨ D)
```

Any propositional formula can be converted to CNF via the **Tseitin transformation**,
which introduces auxiliary variables to avoid exponential blowup. For a formula of
size n, Tseitin produces a CNF of size O(n) that is equisatisfiable (not equivalent)
to the original.

## DPLL: The Foundation

The **Davis-Putnam-Logemann-Loveland** (DPLL, 1962) algorithm is the foundation of
all modern SAT solving:

1. **Unit propagation**: If a clause has only one unassigned literal, that literal
   must be true for the clause to be satisfied. Assign it and propagate.
2. **Pure literal elimination**: If a variable appears only positively (or only
   negatively) across all unresolved clauses, assign it true (or false).
3. **Decision**: Choose an unassigned variable, try both truth values via recursive
   backtracking search.

DPLL is complete but impractical for large formulas without the optimizations
described below.

## CDCL: Conflict-Driven Clause Learning

**Conflict-Driven Clause Learning** (CDCL) extends DPLL with clause learning from
conflicts, transforming SAT solvers from 1960s theorem provers into industrial
verification engines. Core components:

### Watched Literals

The **two-watched-literal scheme** (Moskewicz et al., Chaff 2001) reduces unit
propagation from O(n·m) to amortized O(1) per assignment. Each clause maintains
two "watched" literals. When a variable is assigned, only clauses watching that
literal are checked:

- If the other watched literal is true → clause satisfied, do nothing
- If the other watched literal is unassigned → find a new unassigned literal to watch
- If both watched literals are false and no alternative exists → **conflict**

This is the single most important optimization in CDCL — it enables propagation
through formulas with millions of clauses in microseconds.

### 1st UIP Conflict Analysis

When propagation reaches a conflict, the solver analyzes the **implication graph**
to derive a **learned clause** that prevents the same conflict from recurring.

The implication graph traces which assignments caused which propagations. The
**First Unique Implication Point** (1st UIP) is the dominator node closest to the
conflict on the path from the decision literal. The learned clause is formed by
resolving reason clauses backward from the conflict to the 1st UIP.

The 1st UIP scheme (Zhang et al., 2001) produces shorter, more reusable clauses
than earlier approaches that learned the decision literal's negation.

### VSIDS: Variable State Independent Decaying Sum

**VSIDS** (Moskewicz et al., Chaff 2001) is the dominant decision heuristic. Each
variable has an activity score, incremented (bumped) whenever the variable appears
in a learned clause. An Exponential Moving Average (EMA) decay factor (typically
0.95) periodically reduces all scores. Variables with the highest scores are chosen
as decisions.

Key properties:
- **State independent**: scores don't depend on the current assignment, only on
  conflict history — robust across restarts
- **Exponential decay**: recent conflicts dominate, naturally aging out irrelevant
  variables
- **LRB extension** (Learning Rate Based): blends VSIDS with a learning-rate
  estimate for each variable
- **Anti-aging**: variables that haven't participated in recent conflicts receive
  an exponential age penalty, preventing stale high-activity variables from
  dominating decisions

### LBD: Literal Block Distance

**Literal Block Distance** (Audemard & Simon, Glucose 2009) measures clause quality
by counting the number of distinct decision levels among a clause's literals. Low
LBD (≤ 2) indicates a "glue clause" — highly reusable and worth protecting from
deletion. LBD is the primary metric for clause database management.

### Clause Database Reduction

Learned clauses accumulate during search. A **tiered database** manages them:

| Tier | LBD | Policy |
|------|-----|--------|
| Core | ≤ 2 | Never delete |
| Mid | 3–6 | Activity-based, careful deletion |
| Local | > 6 | Aggressive deletion |
| Recent | all | Protected for a window after creation |

Clauses are periodically promoted from recent to their LBD-appropriate tier. When
the database exceeds its size limit, the deletion policy removes the least useful
clauses, prioritizing local-tier with low activity.

### Restart Strategies

Periodically restarting the search (discarding the current assignment trail but
keeping learned clauses) is critical for escaping unfruitful search branches.

- **Luby restart**: restart intervals follow the Luby sequence (1, 1, 2, 1, 1, 2,
  4, ...), which is optimal for unknown search spaces
- **Glucose-adaptive restart**: track a fast and slow Exponential Moving Average
  of conflict rates; restart when the fast EMA exceeds a threshold multiple of the
  slow EMA — indicates rapid progress has stalled

## Mode Switching

Kissat introduced **mode switching** between two search strategies:

- **Focused mode**: aggressive VSIDS with rapid EMA decay (0.95). Exploits the
  current promising variable ordering. Decisions use the binary max-heap.
- **Stable mode**: periodic score recalculation instead of per-conflict bumping,
  with **reluctant doubling** for random decisions at Luby-sequence intervals.
  Explores alternative variable orderings.

Switching occurs at restart boundaries when the current mode's progress indicator
(conflict count or tick count) exceeds a threshold. Thresholds scale with
n² (where n is switch count), letting modes run longer as the solver matures.

## WalkSAT Local Search

**WalkSAT** (Selman, Kautz, & Cohen, 1994) is a stochastic local search algorithm
that operates on the original (non-learned) clauses before CDCL begins:

1. Start from a random or heuristically-chosen assignment
2. Pick a random unsatisfied clause
3. Within that clause, pick a literal to flip using a **make/break score**: the
   number of clauses that would become unsatisfied if this literal is flipped
4. Flip the chosen literal, update clause satisfaction counts
5. Repeat until all clauses satisfied or flip budget exhausted

The score is exponentially weighted (score = 1/CB^breaks, where CB ≈ 2.0) so
literals with fewer breaks are strongly preferred. This is a probabilistic greedy
strategy — it usually picks the best literal but occasionally explores alternatives.

WalkSAT often solves easy instances in microseconds. For hard instances, the best
assignment found is exported to the VSIDS phase cache, warm-starting CDCL with
promising initial polarities.

## Gaussian Elimination for XOR Constraints

XOR constraints (parity constraints like A ⊕ B ⊕ C = 1) are handled natively via
**Gauss-Jordan elimination** over GF(2). This avoids the exponential CNF encoding
of XOR clauses.

A subset of XOR clauses is selected (by size and unassigned-variable count), a
Boolean matrix is constructed, and Gaussian elimination reduces it to row-echelon
form. The reduced matrix yields:
- **Unit XORs**: single-variable parity constraints → direct assignments
- **Contradictions**: rows of the form 0 = 1 → UNSAT
- **New XOR clauses**: reduced constraints with fewer variables

## Inprocessing

**Inprocessing** (Järvisalo et al., 2012) interleaves formula simplification with
search, running at decision level 0 between restarts:

- **Bounded Variable Elimination (BVE)**: if a variable appears in ≤ threshold
  clauses, eliminate it by resolving all pairs of clauses containing the variable
- **Subsumption**: if clause A's literals are a subset of clause B's, B is redundant
- **Self-subsuming resolution**: if clause A with literal x subsumes B with ¬x,
  B can be strengthened by removing ¬x
- **Failed literal probing**: tentatively assign each literal and propagate; if a
  conflict is found, the negation is a unit clause
- **Clause vivification**: for each clause, check if any literal is redundant by
  asserting its negation and propagating

Inprocessing is "cheap" (runs at level 0 with no trail) and often dramatically
reduces formula size before the hard CDCL work begins.

## DPLL(T): Theory Solver Integration

The **DPLL(T)** architecture (Z3, Microsoft Research) layers theory solvers on
top of a CDCL SAT core. The SAT solver treats theory atoms as Boolean variables.
When it finds a satisfying Boolean assignment, theory solvers check whether the
assignment is theory-consistent. If not, the theory solver returns a **theory
lemma** (a clause) that rules out the inconsistent assignment.

This architecture is the foundation of SMT (Satisfiability Modulo Theories) solving
and enables reasoning about arithmetic, arrays, bit-vectors, and other theories
beyond pure Boolean logic.

## MAX-SAT

**Maximum Satisfiability** finds an assignment that maximizes the number (or
weighted sum) of satisfied clauses, even when the full formula is unsatisfiable.
Our solver uses binary search on a weight threshold with relaxation variables.

## References

- Silva & Sakallah. GRASP: A Search Algorithm for Propositional Satisfiability.
  *IEEE Transactions on Computers*, 1999.
- Moskewicz, Madigan, Zhao, Zhang, Malik. Chaff: Engineering an Efficient SAT
  Solver. *DAC*, 2001.
- Zhang, Madigan, Moskewicz, Malik. Efficient Conflict Driven Learning in a
  Boolean Satisfiability Solver. *ICCAD*, 2001.
- Audemard & Simon. Predicting Learnt Clauses Quality in Modern SAT Solvers.
  *IJCAI*, 2009. (Glucose / LBD)
- Biere. CaDiCaL, Lingeling, Plingeling, Treengeling, and Kissat. *SAT Competition*,
  2020–2024.
- Selman, Kautz, Cohen. Noise Strategies for Improving Local Search. *AAAI*, 1994.
  (WalkSAT)
- Järvisalo, Heule, Biere. Inprocessing Rules. *IJCAR*, 2012.
- Een & Sörensson. An Extensible SAT-solver. *SAT*, 2003. (MiniSat)
- Nikolaev & Ravindran. Hyaline: Fast and Transparent Lock-Free Memory Reclamation.
  *PLDI*, 2021.
