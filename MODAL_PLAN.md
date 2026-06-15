# R4 — Modal Logic System Plan

## MANDATORY CONSTRAINTS — READ BEFORE IMPLEMENTING ANYTHING

> [!CAUTION]
> THESE ARE NOT SUGGESTIONS. EVERY PHASE MUST COMPLY OR THE CODE IS REJECTED.

### 1. Memory Allocation: `github.com/xDarkicex/memory` ONLY

Before writing any code, read the full API of `github.com/xDarkicex/memory` at `../xDarkicex/memory`. Every single allocation in this package MUST use one of:

| Allocator | Use for |
|-----------|---------|
| `memory.Pool` via `MustPoolSlice[T]` | Variable-length slices (formula lists, world paths, tableau branches, parser tokens). Bulk `Reset()` between evaluations. |
| `memory.Arena` via `MustArenaSlice[T]` + `ArenaAppend` | Grow-only data (frame worlds, timeline entries, closure results). `Free()` on teardown. |
| `memory.ShardedFreeList[T]` via `Acquire()` / `Release()` | Fixed-size structs allocated/deallocated at high frequency. The tableau prover creates and destroys thousands of `PrefixedFormula`, `TableauNode`, and boxed `World` handles per satisfiability check. ShardedFreeList gives O(1) alloc/free with no contention across parallel tableau branches. Use for any type whose size is known at compile time and whose lifetime is short and bounded. |

**ABSOLUTELY FORBIDDEN:**
- `make([]T, ...)` — use `memory.MustPoolSlice[T](pool, len, cap)` or `memory.MustArenaSlice[T](arena, len)`
- `make(map[K]V)` — allowed ONLY for small lookup maps (≤ 50 entries, bounded by frame size). All backing slices within maps must be Pool/Arena.
- `new(T)` or `&T{}` for any slice-backed struct — use Pool/Arena for the backing arrays.
- `new(TableauNode)`, `&TableauNode{}`, `new(PrefixedFormula)`, `&PrefixedFormula{}` — these are fixed-size, high-churn structs. Use `ShardedFreeList[TableauNode].Acquire()` / `ShardedFreeList[PrefixedFormula].Acquire()`. This is NOT optional. The GC must never trace tableau expansion data.
- `append()` on a non-Pool/non-Arena slice — use `memory.ArenaAppend` or manual Pool expansion.

Every file that allocates must import `"github.com/xDarkicex/memory"`. The Go GC must never scan modal package data.

### 2. Cyclomatic Complexity: STRICT ≤ 10

Every function, exported or unexported, must have CC ≤ 10. No exceptions. If a function approaches 10, split it. Use `gocyclo` or manual counting (1 + if/for/case/&&/|| count).

Hot-path functions (evaluators, tableau expansion, temporal operators) should target CC ≤ 6. Extract helpers aggressively.

### 3. Testing: 100% Coverage, Every Function

- Every exported function must have at least one dedicated test case.
- Every internal helper must be exercised through exported function tests.
- Run `go test ./modal -cover` after every phase. Target: 100% statement coverage.
- Run `go test ./modal -race` after every phase. Must pass.
- Test edge cases: empty frame, single world, cyclic accessibility, contradictory valuation, maximum-depth recursion.

### 4. Go Doc Comments: Every Public API

Every exported type, function, method, and constant must have a Go doc comment starting with the name:

```go
// World is a handle to a possible world in a Kripke frame.
// Worlds are allocated from an Arena and indexed by uint32 handles.
type World uint32

// Eval evaluates a modal formula at the given world in the model.
// Returns the truth value of the formula, which may be fuzzy if the model
// uses weighted accessibility relations via fuzzy_bridge.go.
func (m *Model) Eval(formula Formula, world World) (TruthValue, error) {
```

No bare exports. Every `func`, `type`, `const`, `var` that is capitalized gets a doc comment.

---

## Design Goals

- **Zero heap allocations** — all backing memory via `github.com/xDarkicex/memory`
- **Cyclomatic complexity ≤ 10** per function (STRICT, enforced)
- **100% unit test coverage** — every exported function, every internal helper
- **Go doc comments** — every public API fully documented
- **Big O documented per function** — time and space
- **Direct integration with daemon's causal graph** — why_ids/how_ids/hop_targets are accessibility relations

---

## Why Modal Logic for Agentic Memory?

The daemon (`libravdbd`) already has a graph: memory nodes linked by `why_ids` (upstream causal), `how_ids` (downstream procedural), and `hop_targets` (undirected association). This graph is a **Kripke frame**.

| Graph concept | Modal logic equivalent |
|---------------|----------------------|
| Memory node | Possible world w |
| why_ids edge | Accessibility relation R_causal |
| how_ids edge | Accessibility relation R_procedural |
| hop_targets edge | Accessibility relation R_association |
| Elevated/guided memory | □p — necessarily true |
| Regular retrieved memory | ◇p — possibly relevant |
| Hop expansion (`etaWhy=0.7`) | Accessibility strength (weighted frame) |
| Memory A contradicts memory B | ¬(□p ∧ □¬p) — consistency constraint |

**What modal logic adds that the daemon doesn't have today:**

1. **Consistency verification:** Before returning top-8, check that no two retrieved memories logically contradict. The SAT solver already exists — modal logic provides the □/◇ operators that frame the verification query.

2. **Belief revision:** When a new elevated memory (□p) contradicts an old belief (◇p), the system knows to demote the old belief to ◇(p ∧ ¬□p) — "it was possible, now it's uncertain." Currently the daemon just overwrites.

3. **Temporal reasoning:** Session timelines are linear temporal frames. "This was true yesterday" → □(yesterday → p). "This might still be true" → ◇p. The compaction system uses this to decide what to summarize vs preserve verbatim.

4. **Multi-agent delegation:** When agent A delegates to agent B, B's memory state is a world accessible from A. B knows p → □_B p. A knows B knows p → □_A □_B p. This is critical for nested delegation chains.

5. **Counterfactual recall:** "What would the agent have retrieved if it searched for X instead of Y?" This is modal possibility semantics over the retrieval function itself. The fuzzy engine scores relevance; modal logic verifies the counterfactual structure.

---

## Reuse from Existing Packages

The modal package must NOT reimplement what already exists in this repo.

| Capability | Already in | How modal uses it |
|------------|-----------|-------------------|
| AND, OR, NOT, XOR, XNOR, NAND, NOR, Implies, Iff | `classical/gates.go` — `Gate` interface | Propositional subformulas within modal expressions. `implies(a,b)` delegates to `classical.Implies(a,b)` |
| Expression parser/lexer | `classical/parser.go`, `classical/lexer.go` | Reuse tokenizer pattern (Pool-backed `[]Token`) for modal formula parsing |
| SAT solver (CDCL) | `sat/cdcl.go` — First-UIP, LBD, VSIDS, XOR, inprocessing | Boolean satisfiability checks inside the tableau prover; equivalence checks for formula simplification |
| CNF conversion (Tseitin) | `sat/cnf_converter.go` | Convert propositional subformulas to CNF for SAT-backed validity checking |
| Truth values [0,1], t-norms, t-conorms, implications | `fuzzy/operators.go` | Phase 6 fuzzy-modal bridge: weighted accessibility relations, fuzzy □/◇ evaluation |
| Defuzzification, membership functions | `fuzzy/` | Fuzzy-modal bridge output crispification |
| `core.LogicSystem` interface | `core/interfaces.go` | Phase 7 `ModalLogicSystem` adapter |
| `memory.Pool`, `memory.Arena`, `memory.ShardedFreeList` | `github.com/xDarkicex/memory` | All allocations |

> **Key insight from Spot:** Spot bundles a SAT solver (PicoSAT) and BDD library (BuDDy) for boolean reasoning. Our SAT solver is more feature-rich than PicoSAT (we have Gaussian elimination, XOR reasoning, inprocessing). What Spot adds that we don't have is BDDs — O(1) equivalence checking after construction. For R4, SAT-backed checks are sufficient. BDDs are a potential R5 optimization.

---

## Mathematical Foundations (from Spot, Somenzi & Bloem, and LTL literature)

These are mathematical identities and structural principles — not copyable code. Every formula below is standard modal logic, documented here to guide our implementation.

### 1. Operator Duality

Every temporal operator has a dual via negation. This is the foundation of NNF conversion.

```
¬□p ≡ ◇¬p          ¬◇p ≡ □¬p          ¬○p ≡ ○¬p
¬(p U q) ≡ (¬p R ¬q)   ¬(p R q) ≡ (¬p U ¬q)
¬(p W q) ≡ (¬p M ¬q)   ¬(p M q) ≡ (¬p W ¬q)
```

These identities mean we only need to implement one of each dual pair natively; the other is derived.

### 2. Negative Normal Form (NNF)

Push all negations inward until they only appear before atomic propositions. This is the canonical representation for all further processing (tableau expansion, model construction, simplification).

```
NNF(¬(p ∧ q)) = NNF(¬p) ∨ NNF(¬q)    NNF(¬□p) = ◇NNF(¬p)
NNF(¬(p ∨ q)) = NNF(¬p) ∧ NNF(¬q)    NNF(¬◇p) = □NNF(¬p)
NNF(¬¬p) = NNF(p)                     NNF(¬○p) = ○NNF(¬p)
NNF(¬(p U q)) = NNF(¬p) R NNF(¬q)
```

### 3. Basic Simplification Rewrites (Spot Level 1)

Applied iteratively to fixed-point on every formula before tableau expansion. These prevent state explosion by reducing formula size BEFORE construction.

**Idempotence:**
```
p ∧ p → p          p ∨ p → p          □□p → □p          ◇◇p → ◇p
```

**Distributivity of modal operators:**
```
□p ∧ □q → □(p ∧ q)     ◇p ∨ ◇q → ◇(p ∨ q)
```

**Absorption and complement:**
```
p ∧ (p ∨ q) → p        p ∨ (p ∧ q) → p
p ∧ ¬p → ff            p ∨ ¬p → tt
p ∧ tt → p             p ∨ ff → p
p ∧ ff → ff            p ∨ tt → tt
```

**Constant propagation under modalities:**
```
□(tt) → tt             ◇(ff) → ff
```

### 4. Syntactic Implication (Somenzi & Bloem 2000)

Fast structural checks for `f → g` without SAT/automata. These rules detect implication from formula shape alone. Used during simplification to eliminate redundant subformulas.

Core rules (expressed as "if f has shape X and g has shape Y, f → g"):
```
p → (p ∨ q)                     // disjunction introduction
(p ∧ q) → p                     // conjunction elimination
f → g and f → h ⟹ f → (g ∧ h)  // conjunction combination
f → h and g → h ⟹ (f ∨ g) → h  // disjunction combination
□p → p                          // reflexivity (T axiom)
□p → □□p                        // transitivity (S4)
◇p → □◇p                        // symmetry (S5)
```

### 5. Array-Backed Graph Storage (Spot's `digraph` pattern)

The single most important performance insight from Spot. States and edges are stored in flat contiguous arrays, not linked objects. This eliminates pointer chasing and enables cache-line-friendly iteration — critical for tableau expansion which visits thousands of states.

**Spot's C++ layout:**
```
state_vector states_;    // std::vector — all states in one contiguous block
edge_vector  edges_;     // std::vector — all edges in one contiguous block
                         // edges_[i].next_succ chains outgoing edges within the array
```

**Our Go mapping:**
```
Frame.Worlds          → Arena via MustArenaSlice[World]     // grow-only world list
Frame.Relations       → Pool via MustPoolSlice[Edge]         // CSR-like edge array
                                           // Edge.Src, Edge.Dst, Edge.NextSucc are indices
TableauNode struct    → ShardedFreeList[TableauNode]         // fixed 64B, per-branch alloc/free
PrefixedFormula struct → ShardedFreeList[PrefixedFormula]    // fixed 16B, per-expansion alloc/free
```

**Why this matters:** During tableau expansion, each branch creates hundreds of PrefixedFormula structs (world + formula pairs). With ShardedFreeList, each is O(1) acquire/release with no GC scan. The edge array is CSR-like: iterate outgoing edges by following `next_succ` indices within a flat Pool slice — zero pointer dereferences.

### 6. Formula Hierarchy (Safety/Guarantee/Obligation/Recurrence/Persistence)

Spot classifies LTL formulas into a hierarchy. This is useful for selecting the optimal proof strategy.

| Class | Shape | Example | Proof strategy |
|-------|-------|---------|----------------|
| **Safety** | □p (something bad never happens) | □¬error | Invariant checking — find a counterexample prefix |
| **Guarantee** | ◇p (something good eventually happens) | ◇success | Reachability — find a path to p |
| **Obligation** | Safety ∨ Guarantee | □safe ∨ ◇goal | Split into two sub-proofs |
| **Recurrence** | □◇p (p holds infinitely often) | □◇alive | Cycle detection — find a reachable cycle containing p |
| **Persistence** | ◇□p (p eventually always holds) | ◇□stable | Find a reachable SCC where p holds everywhere |

The daemon's temporal queries map naturally: "was this true before compaction?" → Safety. "Will this memory still be relevant?" → Guarantee. "Consistency across all retrievals" → Persistence.

### 7. Structural Reduction Before Construction

Spot's single most important algorithmic principle: **aggressively rewrite and simplify the formula before building any graph structure.** This is the lesson from decades of model checking: structural simplification prevents state explosion that no amount of hardware can overcome.

Concretely, for every formula entering our `semantics.go` evaluator or `tableau.go` prover:
1. Convert to NNF
2. Apply basic rewrites to fixed-point
3. Apply syntactic implication to detect redundant subformulas  
4. Classify the formula (safety/guarantee/etc.) to select optimal evaluation strategy
5. Only THEN evaluate or expand

This pipeline is the difference between evaluating a 5-world frame and evaluating a 500-world frame. The rewrites are O(n) on formula size; the evaluation is O(2^w) on world count.

---

## Advanced Techniques from Spot

These are deeper mathematical and algorithmic insights beyond the basic rewrite rules. Each is a standard technique from the model checking literature, extracted from Spot's implementation.

### 8. Formula Hash Consing (O(1) Equality)

Spot's `formula` class uses an internal hash cons table. Every `formula::ap()`, `formula::unop()`, `formula::binop()` returns a canonical pointer. Two formulas are equal iff they point to the same object. This gives:

- **O(1) equality comparison**: pointer comparison, not tree traversal
- **O(1) hashing**: pointer hash or integer formula ID
- **Automatic DAG sharing**: identical subformulas are automatically shared

For our Go implementation, use a global registry keyed by formula structure hash. When constructing a formula, check the registry first. If an identical formula already exists, return the existing reference. This eliminates redundant formula objects entirely — a `□(p ∧ q)` that appears in 50 rules is stored once.

### 9. The `mospliter` Classification System

Spot's simplifier pre-classifies subformulas into categories using bit flags before applying rewrite rules. This enables O(1) "does this subformula have property X?" checks.

| Flag | Meaning | How we compute it |
|------|---------|-------------------|
| `Split_Event` | Purely eventual formula (◇-like) | Syntactic: `◇p`, `p U q` where `q` is eventual |
| `Split_Univ` | Purely universal formula (□-like) | Syntactic: `□p`, `p R q` where `q` is universal |
| `Split_EventUniv` | Both eventual AND universal | `□◇p` is both |
| `Split_Bool` | Pure propositional (no modalities) | Delegates to `classical/` evaluator |
| `Split_GF` / `Strip_GF` | `□◇p` pattern (recurrence) | Match `□◇X` shape, strip outer `□◇` for factoring |
| `Split_FG` / `Strip_FG` | `◇□p` pattern (persistence) | Match `◇□X` shape, strip outer `◇□` for factoring |
| `Split_U_or_W` | `p U q` or `p W q` shape | Binary until/weak-until |
| `Split_R_or_M` | `p R q` or `p M q` shape | Binary release/strong-release |

**High-impact rewrite that depends on classification:**
```
If f is purely eventual:   F(f) = f,  f U g = g,  f M g = f ∧ g
If f is purely universal:  G(f) = f,  f R g = g,  f W g = f ∨ g
If f is both:              X(f) = f   (next is redundant)
```

These three rules alone eliminate entire classes of redundant modal operators.

### 10. Couvreur's On-the-Fly Emptiness Check

The gold standard algorithm for checking ω-automaton emptiness. Applies directly to our tableau prover: satisfiability ≡ non-emptiness of the corresponding automaton.

**Algorithm (pseudocode):**
```
couvreur_check(aut):
    root = empty stack of (index, acc_set)  // SCC roots
    arc  = empty stack of acc_set            // edge acceptance between SCCs
    h = map: state → int                     // 0 = unvisited, -1 = dead
    todo = DFS stack of (state, iterator)
    num = 1

    push initial state with order 1

    while todo not empty:
        state = todo.top()
        if has_next_successor(state):
            edge = next_successor(state)
            dest = edge.dst
            if h[dest] == 0:          // new state discovered
                num++ ; h[dest] = num
                root.push(num, {})
                arc.push(edge.acc)
                todo.push(dest)
            elif h[dest] == -1:        // dead SCC, skip
                continue
            else:                      // back/cross edge — merge SCCs
                threshold = h[dest]
                while threshold < root.top().index:
                    edge.acc |= arc.top()
                    edge.acc |= root.top().condition
                    arc.pop() ; root.pop()
                root.top().condition |= edge.acc
                if accepting(root.top().condition):
                    return true        // found accepting SCC
        else:
            todo.pop()
            if root.top().index == h[state]:  // state is SCC root
                mark all states in this SCC as dead (h[s] = -1)
                root.pop() ; arc.pop()

    return false
```

**Key properties:**
- **On-the-fly**: states are explored lazily, not all need to be visited
- **SCC-based**: tracks strongly connected components via Tarjan-like root stack
- **Dead state pruning**: once a non-accepting SCC is fully explored, all its states are marked dead
- **Two variants**: works with both explicit graphs (unsigned indices, vector storage) and abstract graphs (state pointers, hash maps)

**For our tableau prover:** Each tableau branch is a state. Branch expansion rules are successors. A closed branch is a dead state. An open, completed branch with no contradictions is an accepting SCC → formula is satisfiable.

### 11. Acceptance Mark Bitmask

Spot represents acceptance set membership as a compile-time-sized bitmask. Each edge carries a `mark_t` (bitmask of acceptance sets). This is far more memory-efficient than storing slices of set indices.

For our implementation:
```go
type Mark uint64  // up to 64 acceptance sets per automaton

const (
    Inf0 Mark = 1 << iota  // acceptance set 0
    Inf1                   // acceptance set 1
    // ...
)
```

Acceptance conditions are stored in reverse Polish notation as a bytecode array:
```go
type AccOp uint8
const (
    AccAnd AccOp = iota  // ∧
    AccOr                // ∨
    AccInf               // Inf(mark) — must visit infinitely often
    AccFin               // Fin(mark) — must visit finitely often
)

type AccCondition []AccWord  // RPN bytecode
```

This enables fast evaluation during emptiness checks: push/pop marks on a stack as the bytecode is interpreted.

### 12. Minato's ISOP (Irredundant Sum-of-Products)

For converting BDDs back to minimized Boolean formulas. Useful when our SAT solver returns a model that needs to be expressed as a minimal propositional formula.

**Algorithm sketch:**
```
ISOP(f):
    if f == 0: return 0
    if f == 1: return 1
    x = top_variable(f)
    f0 = f restricted to x=0
    f1 = f restricted to x=1
    g0 = ISOP(f0 - f1)   // cubes in f0 but not f1
    g1 = ISOP(f1 - f0)   // cubes in f1 but not f0
    h  = ISOP((g0 | g1) ∧ (f0 ∧ f1))  // shared cubes
    return (x ∧ g1) ∨ (¬x ∧ g0) ∨ h
```

While we don't have BDDs in R4, this algorithm is worth documenting for R5 if we add BDD support.

### 13. Cut-Point (Articulation Point) Relabeling

Spot's approach to safely renaming Boolean subexpressions for optimization:

1. Build undirected graph from formula syntax tree (parent-child edges)
2. Compute articulation points using Hopcroft-Tarjan: node `v` is an articulation point if `low[v] >= num[u]` where `u` is the parent of `v`
3. Only rename Boolean subformulas that are articulation points AND whose children are also articulation points

This preserves semantic dependencies. For example, `(a ∧ b) U (a ∧ ¬b)` should NOT be relabeled as `p0 U p1` because `a` and `b` are shared across the two sides.

### 14. Eventuality/Universality Propagation

Precompute two boolean properties for every subformula once, then use them in O(1) lookups:

```
is_eventual(p)   = false (atomic)
is_eventual(◇f)  = true
is_eventual(□f)  = is_eventual(f)
is_eventual(f∧g) = is_eventual(f) || is_eventual(g)
is_eventual(f∨g) = is_eventual(f) && is_eventual(g)

is_universal(p)  = false (atomic)
is_universal(□f) = true
is_universal(◇f) = is_universal(f)
is_universal(f∧g) = is_universal(f) && is_universal(g)
is_universal(f∨g) = is_universal(f) || is_universal(g)
```

These are computed bottom-up in O(n) on formula size. They power the high-impact rewrite rules in section 9.

### 15. Formula Splitting by Independent APs

Spot splits conjoined formulas into independent components before performing expensive checks:

```
split_independent_conjunctions(f1 ∧ f2 ∧ ... ∧ fn):
    build undirected graph where:
        nodes = subformulas f1...fn
        edge(fi, fj) if AP(fi) ∩ AP(fj) ≠ ∅
    run connected components
    return one conjunction per component
```

Each component is checked independently — if any is unsatisfiable, the whole conjunction is. For a disjunction, any satisfiable component makes the whole satisfiable. This decomposes a potentially exponential check into smaller independent ones.

---

## SAT Solver Improvements from Spot

While our CDCL solver already exceeds PicoSAT, Spot teaches several complementary techniques:

### B.1 BDD for Boolean Subformulas

Spot converts the Boolean skeleton of temporal formulas to BDDs for O(1) equivalence checking. Our equivalent: bridge `modal/boolean.go` to `sat/cdcl.go` — convert propositional subformulas to CNF via Tseitin, check equivalence with two SAT calls.

For R4, SAT-backed checks are sufficient. For R5, consider adding ROBDD support for the Boolean fragment to get O(1) repeated equivalence queries.

### B.2 Acceptance-Driven Clause Learning

Spot's emptiness check learns which SCCs are "dead" (non-accepting). Our SAT solver could adopt a similar pattern: when a decision level's assignment space is fully explored and unsatisfiable, mark that level's variable subspace as "dead" to avoid redundant exploration in restarts.

### B.3 Formula Structure-Guided Variable Ordering

Spot classifies subformulas by type (eventual, universal, Boolean). Our SAT solver's VSIDS heuristic could weight variables based on the modal structure of the original formula — variables from eventual subformulas get higher activity bias (they're more likely to be decision-critical).

### B.4 Independent Component Decomposition

Before SAT solving, split the CNF into independent variable components (same connected-components algorithm as section 15). Solve each component independently. If any component is UNSAT, the whole formula is UNSAT. This avoids exponential blowup from unrelated clauses interacting in the solver.

---

## Verification Pipeline Architecture from Spot

Spot chains algorithms into a pipeline: **simplify → translate → post-process → check**. Each stage has multiple strategies selected by cost/benefit. These patterns apply directly to our modal package.

### 16. LTL Splitting: Obligation / Suspendable / Rest

Spot's most important decomposition (from `translate.cc`). Before translating a formula to an automaton, it classifies subformulas into three groups:

| Group | Property | Translation strategy | Modal analog |
|-------|----------|---------------------|-------------|
| **Obligation** | `is_syntactic_obligation()` | Translate to minimal WDBA (weak deterministic Büchi) | Safety formulas: `□p`, `p W q` — use direct model checking |
| **Suspendable** | `is_eventual() && is_universal()` | Translate separately, product with "suspension" | `□◇p` — both always-eventual and eventually-always |
| **Rest** | Everything else | Full Couvreur FM construction | General LTL: `p U q`, `GFa → GFb` |

**Key identity:**
```
F(q ∨ u ∨ f) = q ∨ F(u) ∨ F(f)   // eventual split
G(q ∧ e ∧ f) = q ∧ G(e) ∧ G(f)   // universal split
```

This decomposition is the difference between a 10-state automaton and a 10,000-state one.

### 17. Post-Processing Cascade: Cheap First, Then Exact

Spot's `postprocessor::run()` chains algorithms from cheapest to most expensive:

```
1. Simplify acceptance condition              (O(n), always)
2. Remove alternation                         (O(n), always)
3. SCC filter (remove useless states)         (O(n), always)
4. WDBA minimization attempt                  (O(n log n), level-dependent)
5. Simulation-based reduction                 (O(n²) BDD ops, level-dependent)
6. Determinization (Safra/Piterman)           (O(2^n log n), only if Deterministic)
7. SAT-based minimization                     (NP-complete, only if enabled)
8. Compare multiple results, pick smallest    (O(n), always)
```

**Our modal pipeline should follow the same pattern:**
```
1. Syntactic checks:   is_valid_syntactically(f) → trivial true/false
2. Cheap checks:       is_valid_in_frame(f, small_frame) → quick model check
3. Exact check:        tableau_prove(f) → definitive answer
4. Model extraction:   extract_countermodel(f) → if invalid, show why
```

### 18. The "Now / Next" Decomposition (Couvreur FM)

The core of Spot's LTL→automaton translation. Every temporal formula is decomposed at state `s` into three parts:

```
f at state s ≡
  NOW(f, s)      // Boolean condition that must hold at s
  ∧ ○NEXT(f, s)   // What must hold at s's successors (X-formulas)
  ∧ PROMISE(f, s)  // Acceptance promises (what must eventually happen)
```

**The BDD encoding uses three disjoint variable sets:**
- `var_set`: Atomic propositions (what's true NOW)
- `next_set`: X-subformulas (what's NEXT)
- `a_set`: Acceptance promises (what must EVENTUALLY happen)

**Promise rules:**
```
PROMISE(a U b)  = PROMISE(b)        // to satisfy a U b, eventually b
PROMISE(F a)    = PROMISE(a)        // to satisfy F a, eventually a
PROMISE(a M b)  = PROMISE(a ∧ b)    // to satisfy a M b, both must hold
```

This is essentially a **symbolic tableau**: the BDD variables encode tableau branches implicitly, and the Minato ISOP algorithm extracts minimal disjuncts.

### 19. Simulation-Based State Reduction

Spot computes **direct simulation** (suffix inclusion) and **reverse simulation** (prefix inclusion), then iterates to a fixpoint. States that simulate each other are merged; transitions are pruned via implication.

**Core technique (BDD signatures):**
```
signature(s) = ∏ { cond × bdd_var(class_of(dst)) : for each edge s → dst }
```

Two states are equivalent iff they have identical signatures. This is computed by assigning each partition class a BDD variable, then building the signature as a BDD conjunction.

**For modal logic:** This is bisimulation contraction. Two worlds `w1, w2` in a Kripke model are bisimilar (and thus satisfy the same modal formulas) iff they have the same valuation and the same pattern of accessible worlds. The signature computation directly implements this.

### 20. Level-Based Degeneralization

Converts n acceptance sets (generalized Büchi) to 1 set (classic Büchi). The level counter tracks "which acceptance set are we expecting next?"

**Algorithm:**
```
State = (original_state, level)
level ∈ {0, 1, ..., n-1, accepting}

For edge s --cond,{sets}--> d:
    next_level = advance(level, sets)
    if next_level == accepting:
        edge is ACCEPTING
        next_level = 0
    add edge (s,level) --cond--> (d,next_level)
```

**Optimizations:**
- **Skip levels:** If sets {0,1,3} are seen, jump directly to level 3
- **Per-SCC orders:** Different SCCs can use different acceptance set orderings
- **Bottommost copy removal:** Redundant (state, level) copies are merged

**For modal logic:** When a formula has multiple `◇` (eventuality) subformulas, degeneralization tracks which ones have been satisfied. This maps to **progress measures** in tableau-based modal provers.

### 21. Stutter Invariance

A property is stutter-invariant if duplicating states (without changing atomic propositions) doesn't change truth. Formally, for any word `w = x·a·y`, stuttering `w' = x·a·a·y` preserves membership.

**Why this matters:** Stutter-invariant formulas can be checked on **reduced models** where stuttering states are collapsed. This is critical for the daemon: memory retrieval that doesn't change the retrieved set between two time steps can "stutter" — the temporal formula shouldn't care.

**Syntactic check:** A formula without `X` (next) is trivially stutter-invariant. Formulas with `X` only under `□` or `◇` may also be stutter-invariant.

**For our `temporal.go`:** Add `IsStutterInvariant(f Formula) bool` — syntactic check that enables model compression before evaluation.

### 22. DFA Minimization via Hopcroft Partition Refinement

Spot minimizes deterministic automata using a variant of Hopcroft's algorithm with BDD signatures:

```
Hopcroft(states, final, non_final):
    partition = [final, non_final]
    while partition changes:
        for each block B in partition:
            for each state s in B:
                signature(s) = ∏ { cond × bdd_var(class_of(dst)) }
            split B by signature
    return automaton from partition
```

**For modal logic:** This is the algorithm for computing the **coarsest bisimulation** on a labeled transition system. It produces the minimal bisimilar model — the canonical form of a Kripke structure.

### 23. Reactive Synthesis (Constructive Satisfiability)

Spot can synthesize a controller from an LTL specification: given `φ(inputs, outputs)`, produce a strategy that guarantees `φ`.

```
synthesis(φ):
    1. Split APs into inputs / outputs
    2. Translate φ to deterministic parity automaton (DPA)
    3. Convert DPA to parity game (environment vs. controller states)
    4. Solve parity game (Zielonka's recursive algorithm)
    5. Extract winning strategy as Mealy machine
```

**For modal logic:** This is **constructive model extraction**. Given a formula `φ`, not just check if it's satisfiable — build the model (strategy) that witnesses it. Our tableau prover's `ProveSatisfiable(formula) (*Model, bool)` already does this for propositional modal logic. For temporal logic, the parity game approach extends this to infinite paths.

---

## LTSmin: PINS, Symbolic Reachability, and Parallel Verification

LTSmin's key innovation is the **Partitioned Next-State Interface (PINS)** — an abstraction that partitions the state vector into transition groups with dependency matrices. Permissive BSD-like license.

### 24. PINS Dependency Matrices

The PINS abstraction partitions transitions into groups that share the same set of state variables they read/write. Four boolean matrices map groups to state vector positions:

| Matrix | Meaning | Use in our solver |
|--------|---------|-------------------|
| `read_matrix` | Which positions group g reads | "Short vector" projection — only pass relevant atoms to subformula evaluator |
| `may_write_matrix` | Which positions group g may write | Track which atoms a transition could change |
| `must_write_matrix` | Which positions group g always writes | Dead variable elimination in SAT encoding |
| `combined_matrix` | Union of read + write | Classical dependency analysis |

**Short vs. Long vectors:** When evaluating a transition group, only the state variables marked in the read matrix are passed (the "short vector"). The next-state function only produces the write columns (the "copy vector"). Unchanged positions are copied directly. This eliminates redundant data movement.

**For modal logic:** Each transition group maps to a subset of the accessibility relation. The read matrix tells which atomic propositions a modal subformula depends on. If `□p` only depends on proposition `p`, transitions that don't write `p` can skip re-evaluating `□p` — the truth value is copied from the source world.

### 25. FORCE Variable Ordering

LTSmin uses the FORCE algorithm (Aloul, Markov, Sakallah) to minimize the "event span" of dependency matrices — the sum over rows of `(last_col - first_col + 1)`. Smaller spans mean tighter BDD variable clustering.

**Algorithm:**
```
FORCE(matrix):
    repeat until convergence:
        1. COG[i] = center_of_gravity(row_i)  // average column index of 1s in row i
        2. For each column j:
           tent[j] = average(COG[i] for all rows i where m[i][j] == 1)
        3. Sort columns by tent[j] ascending
        4. Permute columns to this order
        5. Compute new event span
        6. If span increased: revert and stop
```

**Mathematical intuition:** Variables that appear together in many clauses/hyperedges should be placed nearby in the decision order. The force-based metaphor: each hyperedge exerts a "spring force" pulling its variables toward its center of gravity.

**For our SAT solver:** Run FORCE on the clause-variable incidence matrix to compute an initial static variable order. This gives the VSIDS heuristic a better starting point and biases it toward proven locality patterns.

### 26. Nested DFS with Early Cycle Detection

LTSmin implements Courcoubetis et al.'s nested DFS for LTL model checking with a critical optimization: **Early Cycle Detection (ECD)**.

**Color scheme:**
```
CYAN  = on blue DFS stack (currently exploring)
BLUE  = visited by blue DFS, fully processed children are PINK
PINK  = red DFS completed (no accepting cycle reachable)
```

**Key optimization — ECD:** Check for accepting cycles directly during the blue DFS, not just in the red DFS. When a successor `t` of state `s` is already CYAN and either `s` or `t` is accepting, an accepting cycle is found immediately. This often avoids launching the red DFS entirely.

**All-Red optimization:** When backtracking, if all children of a state are PINK (red-complete), set the parent to PINK without a separate red DFS. Track which stack levels have all-red children via a bitmask.

**For our tableau prover:** NDFS maps directly to branch exploration. CYAN = active branch, PINK = closed branch (contradiction found), BLUE = open branch pending completion. ECD catches contradictions early before exploring the full branch. The accepting cycle is a satisfiability witness — a completed open branch.

### 27. Guard-Based Partial Order Reduction

LTSmin's most sophisticated POR uses dependency matrices to compute stubborn sets. Five matrices encode transition independence:

| Matrix | Meaning |
|--------|---------|
| `nes` (Necessary Enabling Set) | For a disabled group g, groups that must fire to enable g |
| `nds` (Necessary Disabling Set) | For an enabled group g, groups that must fire to disable g |
| `nce` (Not Co-Enabled) | Groups never simultaneously enabled |
| `dna` (Do Not Accord) | Groups whose execution order matters (don't commute) |
| `not_accords` | Groups with commutativity failure |

**Beam search** finds a minimal stubborn set by scoring groups with a heuristic:
```
h(group) = 1 if disabled (cheap to include)
           visibility_factor * ngroups if enabled + visible
           enabled_count * ngroups if excluded (expensive — we want to exclude)
           2 if included (must be selected)
```

**For modal logic:** The commutativity matrix (`dna`) is the key. Two modal subformulas commute if they don't share atomic propositions. `□p` and `◇q` with disjoint APs can be evaluated in any order — they're independent. This enables pruning symmetric tableau branches.

### 28. Saturation Reachability

LTSmin's most sophisticated symbolic algorithm assigns transition groups to levels based on the highest BDD variable they depend on, then applies fixpoints level-by-level:

```
SAT(visited, groups):
    Assign groups to levels by highest BDD variable
    k = 0
    while k < max_levels:
        old = visited
        apply fixpoint at level k only
        if visited == old: k++          // level saturated
        else: k = back_edge[k]          // re-enter lower levels
```

When a level produces new states, saturation **re-enters lower levels** (`back_edge`), because new states at a high level may enable transitions at lower (previously saturated) levels. This is more efficient than BFS because it converges to the global fixpoint without repeatedly visiting already-saturated levels.

**For modal logic:** Saturation maps to **alternating fixpoint computation** for CTL and the modal mu-calculus. `□◇p` is `νX. μY. (p ∨ □Y) ∧ ◇X` — the alternation depth determines the number of levels. Saturation evaluates the inner fixpoint to completion before advancing the outer one.

### 29. Tree-Based State Compression (TreeDBS)

LTSmin compresses state vectors by recursively pairing slots and storing the pairs in a non-resizing hash table. The resulting tree shares common prefixes — states that differ only in a leaf slot share all internal nodes.

**Incremental update:** Given a related state `prev` and a transition group `g`, only the slots affected by `g` (from the write matrix) are updated. Unaffected subtrees are reused. This makes state storage O(delta) instead of O(state_vector_length).

**Satellite bits:** Tree root nodes carry extra bits for algorithm state (e.g., NDFS colors). This avoids a separate color map.

**For modal logic:** When generating candidate worlds during tableau expansion, most variables are copied unchanged between worlds. TreeDBS-like incremental storage means each new world only allocates for the propositions that actually changed.

### 30. Symbolic Reachability with Vector Set Abstraction

LTSmin abstracts over multiple decision diagram backends (Buddy BDDs, Sylvan BDDs, LDDmc, SDD) via a single `vset` interface:

```
vset_next(dst, src, relation)     → image:  {y | ∃x∈src: (x,y)∈rel}  = ◇src
vset_prev(dst, src, rel, univ)    → preimage: {x | ∃y∈dst: (x,y)∈rel}  = ◇⁻¹dst
vset_union, vset_intersect, vset_minus   → Boolean ops on world-sets
vset_least_fixpoint(set, rel)     → μX. set ∪ rel(X)
```

**Key insight:** `vset_next` directly computes `◇φ` (exists successor satisfying φ). `vset_prev` computes `□φ` (all predecessors satisfy φ). The fixpoint operations compute CTL/mu-calculus operators directly on BDD/SDD representations of world-sets.

**For our modal package:** The `vset` interface is the right abstraction layer. Phase 1-2 can use explicit sets (`[]World` slices). R5 can add a BDD or SDD backend behind the same interface without changing any evaluation code.

---

## Formal Foundations: Inductive/Coinductive Duality (Coq, Coupet-Grimal 2002)

The Rocq (Coq) formalization of LTL reveals the mathematical structure that implementations often obscure: temporal operators are fundamentally **fixpoints** over infinite streams.

### 31. Inductive vs Coinductive Operators

This is the most important structural insight. LTL operators divide cleanly into two classes based on their witness type:

| Operator | Fixpoint | Coq type | Witness | Check algorithm |
|----------|----------|----------|---------|-----------------|
| `◇P` (eventually) | Least (μ) | **Inductive** | Finite prefix ending in `P` | Find a reachable state satisfying `P` |
| `P U Q` (until) | Least (μ) | **Inductive** | Finite prefix of `P` ending in `Q` | Find a reachable state satisfying `Q` with `P` on the path |
| `□P` (always) | Greatest (ν) | **CoInductive** | Infinite proof: `P` holds at every step | Show `P` is invariant under the transition relation |
| `P W Q` (weak until) | Greatest (ν) | **CoInductive** | Infinite proof: either `Q` holds, or `P` holds and the property coinduces | Check invariant preservation |
| `□◇P` (infinitely often) | ν then μ | `always(eventually(P))` | For every step, a finite witness of `P` eventually | Nested DFS |
| `◇□P` (eventually always) | μ then ν | `eventually(always(P))` | Finite prefix to a point from which `□P` holds | SCC-based: find a terminal SCC where `P` is invariant |

**Why this matters for implementation:** Inductive operators can be checked with **finite state space search** (BFS/DFS to find a witness). Coinductive operators require **invariant checking** (prove preservation across all transitions). The tableau method handles both, but understanding which is which guides algorithm selection.

### 32. Formally Proved Theorems

These are theorems proven correct in Coq — they are the mathematical laws of LTL, not implementation details.

**Monotonicity (congruence):**
```
□(P → Q) → (◇P → ◇Q)    // congruence_eventually
□(P → Q) → (□P → □Q)    // congruence_always  (this is the K axiom!)
□(P → Q) → (□◇P → □◇Q)  // congruence_infinitely_often
```

**Reductions:**
```
P U Q → ◇Q               // until_eventually
◇P → (¬P) U P            // eventually_until (requires decidability of P)
◇P → P U ◇P              // until expansion (variant)
```

**Fixpoint properties:**
```
□P → □□P                 // always_idempotence (S4 axiom, proved as theorem)
□P → P                   // holds when the model has no deadlocks
```

**Safety proof rule (invariant induction):**
```
(∀s. init(s) → P(s))  ∧  invariant(P)  →  □P(on_all_runs)
// If P holds initially and is preserved by every step, P holds always
```

**Termination (well-founded measure):**
```
Given well_founded(<) and measure m: state → α:
(∀v. □(A(s) ∧ m(s)=v → (B U (C ∨ (A ∧ m(s) < v)))))
→ (A → B U C)
// If a measure strictly decreases each step until the goal is reached,
// then the goal is eventually reached (no infinite descent)
```

**Duality theorems:**
```
□P ≡ ¬◇¬P               // always = not eventually not
◇P ≡ ¬□¬P              // eventually = not always not
P U Q ≡ ¬(¬Q W (¬P ∧ ¬Q))  // until dual to weak until
```

### 33. The Safety-Progress Hierarchy (Formalized)

Coupet-Grimal's formalization defines the classical hierarchy:

```
invariant(P) = ∀s,t. step(s,t) ∧ P(s) → P(t)     // one-step preservation
safe(P) = ∀str. run(str) → □P(str)                // holds on all runs
leads_to(P,Q) = ∀s,t. P(s) ∧ step(s,t) → Q(t)    // one-step leads-to
fairness(a) = □◇enabled(a) → ◇takes(a)           // weak fairness
fairstr = □◇enabled(fair_step) → □◇takes(fair_step)  // strong fairness on streams
```

### 34. Why This Changes Our Implementation

The inductive/coinductive distinction directly informs our Go code structure:

1. **Inductive operators (`◇`, `U`)**: Implement as **BFS/DFS reachability** in `semantics.go`. These are finite searches — they terminate when a witness is found.

2. **Coinductive operators (`□`, `W`)**: Implement as **invariant checks** in `axioms.go`. These require proof that a property is preserved across all accessible worlds.

3. **Mixed fixpoints (`□◇`, `◇□`)**: Implement via **SCC decomposition** in `tableau.go`. `□◇P` requires finding an SCC where every cycle visits `P`. `◇□P` requires finding a terminal SCC where all states satisfy `P`.

4. **Well-founded termination**: The `wf_leadsto` lemma maps to our `temporal.go` evaluation of `P U Q` with a measure function — if the daemon provides a monotonic measure (e.g., decreasing hop distance), we can prove termination constructively rather than by exhaustive search.

---

## Formula Learning via Skeleton Enumeration (learn_ltl, HSP-IIT)

learn_ltl demonstrates a structured search model for LTL formula synthesis from positive/negative trace examples. Its architecture maps cleanly to Go concurrency patterns.

### 35. Skeleton Tree Enumeration

Instead of blindly generating formula strings, learn_ltl first enumerates **formula shapes** (skeleton trees), then fills them with operators. A skeleton tree of size `s` has `s` leaves (atomic propositions):

```
gen(size):
    if size == 1: return [Leaf]
    unary = gen(size-1) wrapped in UnaryNode
    binary = for each partition left+right = size-1:
               cartesian_product(gen(left), gen(right)) wrapped in BinaryNode
    return unary ∪ binary
```

This separates **structural enumeration** from **operator assignment**. The number of skeletons grows as the Catalan numbers: 1, 1, 2, 5, 14, 42, 132... — manageable for small sizes.

**For Go:** Each skeleton is an independent work unit. A goroutine pool can explore skeletons in parallel, with a channel collecting consistent formulas. This maps the `rayon::par_iter` + `find_any` pattern directly to `errgroup` or a worker pool.

### 36. Normalization via Filtering (Not a Separate Pass)

The key innovation: normalization is **built into generation**, not applied afterward. Each operator constructor has a `check_*` filter:

```
gen_formulae(skeleton, vars):
    Leaf → [Atom(v) for v in vars]
    UnaryNode(child) → for each child_formula in gen_formulae(child):
        if check_not(f):    add Not(f)
        if check_next(f):   add Next(f)
        if check_globally(f): add Globally(f)
        if check_finally(f): add Finally(f)
    BinaryNode(left, right) → cartesian_product × for each:
        if check_and(l,r):    add And(l,r)
        if check_or(l,r):     add Or(l,r)
        if check_implies(l,r): add Implies(l,r)
        if check_until(l,r):  add Until(l,r)
```

A formula that fails its check is never generated. This means the output is **already normalized** — no separate normalization pass needed.

**The ~40 filter rules (documented in learn.rs):**

| Filter | Key rules |
|--------|-----------|
| `check_not` | `¬¬φ→φ`, `¬(φ→ψ)→φ∧¬ψ`, `¬Fφ→G¬φ`, De Morgan redundancies |
| `check_next` | `XFφ→FXφ` (commute, finite trace OK) |
| `check_globally` | `GGφ→Gφ` (idempotence), `GXφ→false` (finite trace) |
| `check_finally` | `FFφ→Fφ` (idempotence) |
| `check_and` | Commutativity (`l<r`), excluded middle, absorption, distribution, `X(φ∧ψ)≡Xφ∧Xψ`, `G(φ∧ψ)≡Gφ∧Gψ`, `(φ₁Uψ)∧(φ₂Uψ)≡(φ₁∧φ₂)Uψ` |
| `check_or` | Commutativity (`l<r`), excluded middle, absorption, `X(φ∨ψ)≡Xφ∨Xψ`, `F(φ∨ψ)≡Fφ∨Fψ`, `(φUψ₁)∨(φUψ₂)≡φU(ψ₁∨ψ₂)`, Until expansion, Finally expansion |
| `check_implies` | `¬φ→ψ≡ψ∨φ`, `φ→¬ψ≡¬(ψ∧φ)`, currying: `φ₁→(φ₂→ψ)≡(φ₁∧φ₂)→ψ` |
| `check_until` | `φUφ≡φ`, `X(φUψ)≡(Xφ)U(Xψ)`, `φU(φUψ)≡φUψ` |

**Commutativity trick:** `check_and` and `check_or` enforce `left < right` (total ordering on `SyntaxTree` via `#[derive(Ord)]`). Only the canonically-ordered variant passes. This breaks `φ∧ψ` vs `ψ∧φ` symmetry without needing a separate commutative normalization pass.

### 37. Smart Trace Evaluation

The `eval_at_time` function has two clever optimizations:

**Reverse iteration for □ and ◇:**
```rust
Globally(branch) => (time..trace.len()).rev().all(|t| branch.eval_at_time(trace, t))
Finally(branch)  => (time..trace.len()).rev().any(|t| branch.eval_at_time(trace, t))
```
Evaluating from the end of the trace backward means shorter suffixes are checked first. "Interpreting on shorter traces is generally faster." For Globally, if the last state fails, it short-circuits immediately without checking the entire trace.

**Until short-circuit:**
```rust
Until(left, right) => for t in time..trace.len():
    if right.eval_at_time(trace, t): return true   // found Q, done
    else if !left.eval_at_time(trace, t): return false  // P failed, done
```
No recursion needed. The loop terminates as soon as either the right side is satisfied or the left side fails.

### 38. Sample-Based Search Model

The search problem: given `positive_traces` (must satisfy formula) and `negative_traces` (must falsify formula), find the smallest formula consistent with both.

```
solve(sample):
    if !sample.is_solvable(): return None   // conflicting traces
    for size in 1..∞:
        if multithread:
            skeletons.par_iter().flat_map(gen_formulae).find_any(is_consistent)
        else:
            skeletons.iter().flat_map(gen_formulae).find(is_consistent)
```

**Size-based iteration** guarantees the smallest formula is found first. The search terminates at the first consistent formula (which is minimal by construction).

**Solvability pre-check:** `is_solvable()` returns false if any positive trace is identical to a negative trace — an inconsistent sample. This is a cheap early exit.

**For Go concurrency:** The search is embarrassingly parallel at each size level. Skeletons are independent; each goroutine explores one skeleton's formula space. A `context.Context` with cancellation stops all workers when any goroutine finds a solution.

### 39. Relevance to the Daemon

This is directly applicable to the daemon's memory recall scoring:

1. **Monitor = atomic proposition.** Each daemon sensor (charge level, room, door state) is a `Monitor` trait returning `bool` — an atomic proposition.
2. **Trace = system execution.** The `run()` loop collects a `Trace<N>` — a sequence of `[bool; N]` arrays, one per time step.
3. **Formula = recall constraint.** A learned LTL formula describes the temporal pattern of a memory: "the agent was in the lab AND eventually reached the charging station BEFORE the battery died."
4. **Satisfaction check = recall match.** `formula.eval(trace)` checks if a memory trace satisfies a recall constraint. This is O(|trace| × |formula|) — fast enough for real-time scoring.
5. **Learning = pattern discovery.** Given positive traces (successful recalls) and negative traces (failed recalls), `solve()` discovers the minimal temporal formula that distinguishes them. This is the daemon learning *why* certain memories are recalled.

**For our `temporal.go`:** The `eval_at_time` pattern maps to `TemporalModel.EvalAtWorld(formula, worldIdx, timeline)`. The reverse-iteration optimization for □/◇ applies directly — evaluate from the last accessible world backward.

---

## Deontic Reasoning Test Matrices (DeonticBench)

DeonticBench provides real-world legal/regulatory test cases across five domains. The value for us is the **problem schemas** — structured patterns of permissions, prohibitions, and obligations that exercise a logic engine's correctness.

### 40. Three Deontic Reasoning Patterns

| Pattern | Domain | Structure | Modal encoding |
|---------|--------|-----------|----------------|
| **Combinatorial optimization** | Airline baggage fees | Choose optimal subset of bags to minimize cost under layered size/weight/class rules | `◇(minimal_cost)` — there exists a bag assignment satisfying all constraints |
| **Multi-statute chaining** | Housing eviction law | Chain rules across statutes: `statute → jurisdiction → refers_to → predicate → answer` | `□(statute(S₁) → ... → answer)` — the law implies the answer |
| **Hierarchical computation** | Tax code | Bottom-up: income → AGI → deductions → exemptions → taxable → brackets → total | Pure function composition: `tax = f(g(h(income)))` |

### 41. Deontic Operator Catalog

The test cases exercise these deontic operators, which are distinct from classical modal □/◇:

| Deontic operator | Natural language | Logical meaning | Tested in |
|------------------|------------------|-----------------|-----------|
| **Permission** | "may", "is permitted", "can file in" | `P(φ) ≡ ¬O(¬φ)` — at least one compliant path exists | Housing (filing options), Airline (complimentary bag choices) |
| **Prohibition** | "may not", "cannot", "excluded" | `F(φ) ≡ O(¬φ)` — no compliant path exists | Airline (oversize limit), Tax (exclusions) |
| **Obligation** | "must", "shall", "required" | `O(φ)` — all compliant paths satisfy φ | Tax (filing requirements), Housing (service methods) |
| **Exception** | "unless", "except", "provided that" | `O(φ) ∧ ¬exception → O(ψ)` — conditional override | Housing (district court operative vs not), Tax (standard vs itemized) |
| **Priority** | "the higher of X vs Y applies" | `max(penalty_X, penalty_Y)` — conflict resolution | Airline (oversize vs overweight = max) |
| **Default** | "if no rule applies, then X" | `¬∃applicable_rule → X` — fallback | Housing (venue default), Tax (standard deduction default) |

### 42. The "Choose Best Combination" Pattern (Airline)

The airline domain encodes a pattern where the correct answer requires exploring a combinatorial space:

```
1. For each bag: compute size penalty = f(dimensions)
2. For each bag: compute weight penalty = g(weight, threshold)
3. For each bag: penalty = max(size_penalty, weight_penalty)
4. Choose N bags as complimentary (base fee = $0)
5. Remaining bags get position-based base fees ($0, $0, $150, $200, ...)
6. Total = ticket + base_fees + penalties
7. Answer = MINIMUM over all choices of which bags are complimentary
```

**This is deontic choice under constraints**: the passenger has permission to designate any bags as complimentary, the airline has the obligation to charge base fees for the rest, and penalties are obligations triggered by exceeding thresholds. The optimal choice minimizes total cost.

**For verification:** This pattern requires the logic engine to correctly handle `max()`, `min()`, threshold cascades (≤62→$0, ≤65→$30, ≤115→$200), and combinatorial search over bag assignments.

### 43. The Statutory Chaining Pattern (Housing)

Legal reasoning chains across multiple statutes to derive a binary answer:

```
Statute 1: "District court has jurisdiction over summary proceedings"
Statute 2: "If district court not operative, municipal court is proper"
Statute 3: "Venue may be changed to any other court"

Question: "Are eviction cases first heard in municipal court?"

Resolution:
    statute_of_state(Law, State)           ← which law applies?
    ∧ jurisdiction(Law, municipal_court)    ← does it grant jurisdiction?
    ∧ refers_to(Law, summary_proceedings)   ← does it cover evictions?
    → answer(yes)
```

**This is deontic implication chaining**: each statute adds a premise, and the question reduces to whether the conjunction of all applicable statutes entails the answer.

**For verification:** This pattern tests whether the logic engine correctly handles:
- Multi-premise conjunction over separately stated rules
- Jurisdiction scoping (which state's law applies)
- Exception handling (primary vs secondary methods, operative vs non-operative courts)
- Binary entailment (yes/no from statute text)

### 44. Relevance to the Daemon

Deontic reasoning matters for the daemon because agent policies are deontic:

1. **Recall scoring as permission**: The daemon *may* return memory M if it satisfies relevance threshold. `P(return(M)) ≡ score(M) > threshold`.
2. **Consistency as obligation**: The daemon *must not* return contradictory memories. `O(¬(return(M₁) ∧ return(M₂) ∧ contradicts(M₁, M₂)))`.
3. **Hop expansion as priority**: When multiple retrieval paths exist, take the one with highest confidence. `max(path₁.confidence, path₂.confidence)`.
4. **Policy chaining**: Agent policies chain: "If in context C, AND memory type is T, AND recency > R, THEN boost score by B." This is the housing statutory chaining pattern applied to memory retrieval rules.

**For the test suite (`modal/modal_test.go`):** DeonticBench's smoke tests (5 cases per domain × 5 domains = 25 cases) provide a ready-made validation suite for deontic reasoning correctness. Each case has a verified Prolog reference implementation and a numeric/binary label.

---

## Four-State Deontic Minimization (Azzopardi, Gatt & Pace 2016)

The ContractFactory in `deontic-logic-with-unknowns` implements algebraic minimization of deontic contracts using a 4-state lattice. This maps directly to our bit-vector operations for zero-overhead rule evaluation.

### 45. The Four-State Deontic Lattice

Every contract evaluates to exactly one of four terminal states. This is a 2-bit encoding:

| State | Bits | Meaning | Algebraic role |
|-------|------|---------|----------------|
| **SAT** (Satisfied) | `00` | Norm fulfilled, no further obligation | Identity element: `C & SAT = C` |
| **VIOL** (Violated) | `01` | Norm breached, irrecoverable | Absorbing zero: `C & VIOL = VIOL` |
| **UNK** (Unknown) | `10` | Insufficient info to determine | Absorbing zero: `C & UNK = UNK` |
| **BASE** (base norm) | `11` | Obligation/Permission/Prohibition — needs action evaluation | The actual deontic content |

**Two absorbing states (VIOL, UNK) and one identity (SAT).** This is a De Morgan algebra with an extra "unknown" value — a 4-valued logic (like Belnap's FDE) applied to deontic reasoning.

### 46. ContractFactory Minimization Identities

Each composition operator reduces to minimal equivalent form using these algebraic laws:

**Concurrent (C₁ & C₂):**
```
C & C = C                          // idempotence
C & SAT = C                        // identity (SAT is neutral)
SAT & SAT = SAT                    // identities compose
C & VIOL = VIOL                    // absorption (violation)
C & UNK = UNK                      // absorption (unknown)
(C₁ & C₂) & C₃ = flatten({C₁, C₂, C₃})  // associativity + dedup via set union
```

**Composed (C₁ > C₂ — sequential):**
```
VIOL > C = VIOL                    // violation in first step propagates
UNK > C = UNK                      // unknown in first step propagates
SAT > C = C                        // (implicit — SAT triggers second step)
```

**Reparation (C₁ *> C₂ — if violated, do reparation):**
```
SAT *> C = SAT                     // primary contract satisfied, reparation skipped
UNK *> C = UNK                     // primary contract indeterminate
VIOL *> C = C                      // REPARATION TRIGGERS — violation activates the secondary
```

**Key insight:** Reparation is the deontic equivalent of try/catch. If the primary obligation is violated, the reparation contract activates. This models real-world penalty clauses: "You must pay by April 15. If you don't, you must pay with a 5% penalty."

### 47. Static Reduction (SyntacticReduction)

Each contract type can determine its state BEFORE evaluating actions, purely from structure:

```
Concurrent.Reduction():
    reduce both children
    if either = VIOL → return VIOL    // violation propagates upward
    if either = UNK → return UNK      // unknown propagates upward
    if either = SAT → return other    // identity collapse
    else → return Concurrent(reduced_child₁, reduced_child₂)

Composed.Reduction():
    reduce first child
    if first = VIOL → return VIOL
    if first = UNK → return UNK
    else → return self

Reparation.Reduction():
    if StartsWith() = SAT → return SAT     // primary already satisfied
    if StartsWith() = UNK → return UNK     // primary indeterminate
    else → return self
```

**This is O(depth) static analysis** — no action evaluation, no state space exploration. The reduction collapses the contract tree to its minimal form using only structural information.

### 48. Bit-Vector Mapping for Zero-Overhead Evaluation

The 2-bit encoding enables bitwise evaluation of deontic contract operations:

```
// 2-bit encoding per contract node
SAT  = 0b00    VIOL = 0b01    UNK  = 0b10    BASE = 0b11

// Concurrent: bitwise AND with absorption table for terminal states
func concurrent(a, b uint8) uint8 {
    if a == VIOL || b == VIOL { return VIOL }
    if a == UNK  || b == UNK  { return UNK }
    if a == SAT  { return b }
    if b == SAT  { return a }
    return concurrent_label(a, b)  // both BASE — encode as concurrent pair
}

// Composed: check first, propagate terminals
func composed(a, b uint8) uint8 {
    if a == VIOL || a == UNK { return a }
    return composed_label(a, b)
}

// Reparation: check primary satisfaction
func reparation(a, b uint8) uint8 {
    if a == SAT { return SAT }
    if a == UNK { return UNK }
    return reparation_label(a, b)  // VIOL or BASE activates reparation b
}
```

**For batch evaluation across multiple rules:** Pack contract states into `uint64` bit-vectors (like `classical/bitvector.go`). A single `AND` instruction evaluates all concurrent contracts simultaneously. Terminal state propagation uses a precomputed lookup table indexed by 2-bit pairs.

**For the daemon:** Policy rules are contracts. "Must have recency > 0.5" = Obligation. "May return low-confidence memories" = Permission. "Must not return contradictory pairs" = Prohibition. Each evaluates to SAT/VIOL/UNK/BASE. The concurrent composition of all active policies gives the final deontic state of the retrieval — SAT = compliant, VIOL = non-compliant, UNK = indeterminate.

### 49. Conflict Analysis via Automaton Exploration

The `ConflictAnalysis` directory builds a transition system over contracts:
- **States** = minimized contract expressions
- **Transitions** = action sets (permutations of all actions)
- **Conflict** = reachable state where a contract transitions to VIOL

This is the deontic equivalent of model checking: `∃ path · contract →* VIOL`. A contract has a potential conflict if some sequence of actions leads to violation.

**For verification:** The `oneStepAwayContracts()` method generates all possible next states. The contract transitions are deterministic given an action set. Conflict analysis is reachability on the contract state graph.

---

## Synchronous Data-Flow and Reactive Constraints (Lustre V4 / Lutin, VERIMAG)

Lustre and Lutin implement the synchronous hypothesis — computation is instantaneous relative to the environment. This maps directly to discrete-time LTL semantics.

### 50. The Synchronous Data-Flow Model (Lustre)

Lustre programs are sets of equations over **infinite streams** of values — one value per clock tick. This is mathematically identical to LTL over discrete time traces.

**Core operators:**

| Lustre operator | Meaning | LTL equivalent |
|-----------------|---------|----------------|
| `pre(x)` | Previous value of x (initially undefined) | `○⁻¹x` (previous) |
| `init -> body` | `init` at t=0, `body` at t>0 | `init ∧ ○(body)` |
| `x = expr` | Equation: x equals expr at every tick | `□(x ↔ expr)` |
| `assert cond` | Verification condition: cond must hold | `□cond` |
| `when(clock, body)` | Activate body only when clock is true | Clock-restricted subformula |

**Key mathematical principle:** A Lustre program `P` with inputs `I`, outputs `O`, and state variables `S` defines a Mealy machine:
```
(S_t, O_t) = f(S_{t-1}, I_t)     // transition function
S_0 = init                        // initial state
```
The compiler extracts this transition function by solving the system of equations at each clock tick. The solution is a fixed-point: all equations must be simultaneously satisfied given the inputs and previous state.

### 51. Clock Calculus and Dependency Analysis

Lustre's type system infers a **clock** for every expression — a Boolean stream indicating when the expression is active. An expression is only evaluated when its clock is true.

**This is the PINS dependency matrix in language design:**
- Each variable has a "read clock" (when is it needed?) and a "write clock" (when is it computed?)
- The clock hierarchy forms a tree: child clocks are refinements of parent clocks
- At each tick, only the active subset of equations needs to be solved

**For our modal evaluator:** When evaluating `□p` at world `w`, if proposition `p` hasn't changed since the last evaluation (the valuation is "clocked" — same truth value), the result is cached. The clock calculus tells us which subformulas need re-evaluation after each world transition.

### 52. The Compilation Chain: Equations → Automata

Lustre's `ec2oc` compiler extracts the finite-state control structure from data-flow equations:

```
Lustre source (.lus)
    → lus2ec (expand arrays, inline nodes, monomorphize)
    → Expanded code (.ec) — single node, flat equations
    → ec2oc (extract Boolean state variables, compute transition relation)
    → Object code (.oc) — finite automaton with:
        - States = valuations of Boolean state variables
        - Transitions = equation satisfaction for each input combination
        - Outputs = computed from state + inputs
    → ocmin (minimize automaton) or poc (generate C code)
```

**The extraction algorithm:** Partition variables into Boolean state variables and others. The Boolean state variables form the automaton states. For each state (valuation of Boolean variables), compute the transition relation: which next states are reachable given which input valuations. This is the same procedure as building a Kripke frame from a modal formula's canonical model.

### 53. Lutin's BDD + Polyhedra Solver

Lutin solves constraints over reactive systems using a **combined Boolean/numeric solver**:

```
is_satisfiable(formula, inputs, state_vars):
    bdd = formula_to_bdd(formula, inputs, state_vars)    // Boolean abstraction
    if Bdd.is_false(bdd): return false                    // fast Boolean UNSAT
    
    n, m = count_solutions(bdd)                           // check numeric feasibility
    if (n, m) == (0, 0): return false                     // Boolean SAT but numeric UNSAT
    return true                                           // actually SAT
```

**Two-phase check:** Boolean satisfiability via BDD (cheap, exact). If Boolean-SAT, check numeric consistency via polyhedral analysis (more expensive). This avoids exploring numerically infeasible Boolean solutions.

**For our SAT solver:** This is the same pattern as our `fuzzy_smt.go` — separate the Boolean skeleton from numeric constraints. Solve the Boolean part first, then check numeric feasibility on the Boolean model.

### 54. Lutin's Temporal Standard Library

Lutin provides a standard library (`temporal.lut`) implementing common LTL patterns as synchronous stream equations:

```
OnceNow(cond)         = cond                        // p holds now
Repeat(body)          = loop body                   // body repeats forever
Always(cond)          = loop cond                   // □cond
Until(event, body)    = assert ¬event in body        // body holds until event
From(event, body)     = loop ¬event fby body         // after event, body holds
AsLongAs(event, body) = assert event in body         // body holds while event
OnceWithin(delay,cond)= loop [0,delay] ¬cond fby cond // ◇_{≤delay} cond
when(cond, body)      = loop { cond→body | ¬cond }  // activate on condition
```

**Key insight:** These are LTL operators expressed as executable stream equations. Each maps to a finite-state automaton constructor. The `loop` operator is □ (always). The `fby` operator is sequential composition. The `assert` operator is a verification condition.

**For our `temporal.go`:** These patterns are directly implementable as temporal formula constructors. `AsLongAs(event, body)` = `□(event → body)`. `Until(event, body)` = `body W event` (weak until — body holds as long as event hasn't occurred). `From(event, body)` = `event → ○(□body)`.

### 55. Application to the Daemon

The synchronous model maps naturally to daemon memory retrieval:

| Lustre/Lutin concept | Daemon mapping |
|---------------------|----------------|
| Clock tick | Retrieval cycle (one evaluation of the recall scoring pipeline) |
| `pre(x)` | Previous retrieval result (cached score from last cycle) |
| `assert cond` | Consistency constraint (must hold at every retrieval) |
| `when(cond, body)` | Conditional retrieval (only evaluate when context matches) |
| `fby` (followed by) | Sequential pipeline stages (filter → score → rank → select) |
| `loop` | Recurring evaluation (every retrieval cycle) |
| Weighted choice `{| w1 |> e1 |}` | Probabilistic recall with confidence weights |

**Timeline as synchronous stream:** The daemon's session timeline IS a Lustre stream. Each time step is a clock tick. The retrieval score is an equation: `score_t = f(pre(score), input_memory, hop_weights)`. Consistency constraints are assertions: `assert ¬(return(M₁) ∧ return(M₂) ∧ contradicts(M₁, M₂))`.

---

## Tsetlin Machine Clause Logic (Graph Tsetlin Machine, Granmo 2024)

Graph Tsetlin Machines learn propositional logic clauses from graph-structured data using Tsetlin Automata — finite-state reinforcement learners. MIT-licensed.

### 56. Tsetlin Automata as Logic Learners

A Tsetlin Automaton (TA) is a finite-state machine with 2N states that learns the optimal Boolean action (include/exclude a literal). The state is stored as an N-bit unsigned integer. The most significant bit determines the action.

**State transition on feedback:**
- **Reward:** Move deeper into current action — state moves away from the decision boundary
- **Penalty:** Move toward the center — state moves toward the decision boundary

This is discrete reinforcement learning with bounded states. Unlike gradient descent, there is no learning rate decay, no vanishing gradients, no floating point — just integer state transitions.

### 57. Clause-Based Pattern Recognition

A clause is a conjunction of literals:
```
Cⱼ(x) = l₁ ∧ l₂ ∧ ... ∧ lₙ    where lᵢ ∈ {xᵢ, ¬xᵢ}
```

Each literal's inclusion is controlled by a TA. The clause output is 1 if all included literals match the input, 0 otherwise. A clause is a **propositional logic rule** — directly human-readable.

**Classification by weighted voting:**
```
class_sum(c) = Σⱼ wⱼ_c · Cⱼ(x)
prediction = argmax_c class_sum(c)
```

Where `wⱼ_c` are integer clause weights learned via feedback. The class with the highest weighted sum of satisfied clauses wins.

### 58. The Learning Rule

Two types of feedback drive learning:

**Type I (clause SHOULD output 1):**
- Clause outputs 0 → Each TA that should include its literal gets reward for including, penalty for excluding
- Clause outputs 1 → All TAs get reward for current action (reinforcement)

**Type II (clause SHOULD output 0):**
- Clause outputs 1 → Included TAs get penalty (they contributed to an incorrect activation)
- Clause outputs 0 → No penalty (already correct)

**Key property:** The learning rule only activates TAs when the clause's output is wrong. Correct clauses are left alone — no unnecessary state changes.

### 59. Hypervector Encoding for Graph Nodes

Graph node properties are encoded as sparse hypervectors:

```
hypervector(symbol) = {bit_pos₁, bit_pos₂}  // small set of positions in a large sparse vector
X[node] = OR over all symbols: set bits at hypervector positions
```

**Why sparse?** The sparse encoding ensures that different symbols have low overlap probability. Two unrelated symbols are unlikely to share bit positions. This enables the clause to learn distinguishability: `prop_a ∧ prop_b` activates different bits than `prop_c ∧ prop_d`.

**For our bit-vector engine:** This maps directly to `classical/bitvector.go`. Each node property is a bitmask. Clause evaluation is bitwise AND across literal masks. The entire clause evaluation is a single SIMD AND instruction.

### 60. Message Passing = Modal Depth

Multi-layer GTMs propagate messages across graph edges:

```
Layer 0: clause_output₀(node) = evaluate_clauses(node_properties)
Layer d: clause_output_d(node) = evaluate_clauses(neighbor_messages)
         where neighbor_messages = aggregate(clause_output_{d-1}(neighbors))
```

**This IS modal logic evaluation at depth d:**
- Layer 0 = Propositional: what's true at this node?
- Layer 1 = ◇-depth 1: what's true at this node's neighbors?
- Layer 2 = ◇-depth 2: what's true at neighbors of neighbors?

The number of message-passing layers equals the modal depth of the formula being evaluated. The clause `neighbor_has_prop_a ∧ neighbor_has_prop_b` is equivalent to `◇prop_a ∧ ◇prop_b` — both neighbors must exist and have those properties.

### 61. Relevance to Our Package

The Graph Tsetlin Machine bridges machine learning and propositional logic. For our codebase:

| GTM concept | Our package mapping |
|-------------|---------------------|
| Clause `Cⱼ(x) = ∧ lᵢ` | `classical.BoolVector.And()` — single SIMD instruction |
| Literal inclusion TA | `sat.CDCLSolver` decision variable — include or exclude |
| Hypervector encoding | `classical.BitwiseInt` — sparse bitmask per atom |
| Message passing depth | `modal.Frame` accessibility — modal depth of evaluation |
| Clause weight `wⱼ_c` | Fuzzy `TruthValue` — weighted belief in a rule |
| Classification voting | `fuzzy.MamdaniEngine` — rule aggregation |
| Learning from examples | `learn_ltl.solve()` — formula synthesis from traces |

**For the daemon:** A GTM could learn the recall scoring function directly from retrieval outcomes. Each clause is a rule like "memory has tag T AND recency > 7 days AND source is elevated → recall score +0.8." The Tsetlin Automata learn which literals matter, and the weights reflect confidence. The entire learned model is a set of propositional clauses — directly verifiable with our SAT solver.

---

## Cognitive Memory Models for the Daemon (Vestige)

Vestige implements production-grade cognitive memory models: FSRS-6 spaced repetition, prediction error gating, synaptic tagging, spreading activation, and dual-strength memory. AGPL-3.0 — extract math only.

### 62. FSRS-6 Power Law Forgetting

The gold standard spaced repetition algorithm (21 parameters trained on 700M+ Anki reviews). Uses a **power law** forgetting curve, not exponential.

**Retrievability (probability of recall):**
```
R(t, S) = (1 + f · t / S)^(-w₂₀)
where f = 0.9^(-1/w₂₀) - 1
```

- `R` = retrievability (0 to 1)
- `t` = time since last review (days)
- `S` = stability (time for R to drop to 90%)
- `w₂₀` = personalized decay parameter (0.1-0.8, default 0.1542)

**Memory stability updates (next_stability):**
```
D' = w₇·D₀ + (1-w₇)·(D - w₆·(G-3)·(10-D)/9)   // difficulty update with mean reversion
S' = S · (e^w₈) · (11-D)^w₉ · (S^w₁₀·R^w₁₁)     // stability increase
```

**Initial values by first review grade:**
```
S₀(Again) = w₀ = 0.212 days
S₀(Hard)  = w₁ = 1.293 days
S₀(Good)  = w₂ = 2.306 days
S₀(Easy)  = w₃ = 8.296 days
```

**For the daemon:** Every memory node in the graph has a `stability` and `difficulty` field. The retrievability `R(t,S)` is the probability that the memory is recallable at time `t`. When the daemon retrieves a memory, it updates the FSRS state based on whether the memory was actually useful (the user's feedback = the "grade"). Memories that are consistently useful get higher stability and are scheduled further apart.

### 63. Dual-Strength Memory Model

Bjork & Bjork (1992): every memory has two independent strengths.

| Strength | Behavior | Daemon field |
|----------|----------|-------------|
| **Storage Strength** | Only increases, never decreases. Reflects total encoding depth. | `storage_strength` |
| **Retrieval Strength** | Decays over time, restored by access. Reflects current accessibility. | `retrieval_strength` |

**Accessibility score (weighted combination):**
```
accessibility = 0.5 × retention + 0.3 × retrieval_strength + 0.2 × storage_strength
```

**Four memory states based on accessibility:**
| State | Accessibility | Meaning |
|-------|---------------|---------|
| Active | ≥ 70% | Immediately retrievable |
| Dormant | 40-70% | Retrievable with effort |
| Silent | 10-40% | Rarely surfaces |
| Unavailable | < 10% | Below threshold |

**For the daemon:** This maps to the `why_id` / `how_id` / `hop_target` edges. Storage strength maps to `why_id` count (causal evidence accumulates). Retrieval strength maps to `hop_target` frequency (recent associations). The dual model explains why a well-established memory can be temporarily hard to recall.

### 64. Spreading Activation Network

Collins & Loftus (1975): Activating one memory node spreads activation to connected nodes. Activation decays with graph distance.

```
activate(node, level):
    node.activation = min(1.0, level)
    if level > threshold:
        for each edge(node, neighbor, weight):
            spread = level × weight × decay_factor
            activate(neighbor, spread)
```

**Parameters:**
- Default decay per hop: 0.7
- Minimum activation threshold for propagation: 0.1
- Max activation: 1.0 (saturates)

**Edge types:** Semantic, Temporal, Spatial, Causal, PartOf, UserDefined.

**For the daemon:** The daemon's `hop_targets` ARE the spreading activation edges. When a memory is recalled, activation spreads along `hop_targets` to related memories. The decay factor (0.7 per hop) determines how far activation reaches. This is mathematically equivalent to fuzzy ◇ evaluation: `◇P` at world `w` computes max truth of `P` across all `R(w)`-accessible worlds, weighted by edge strength.

### 65. Synaptic Tagging & Capture (Retroactive Importance)

Frey & Morris (1997): Weak stimulation creates a temporary "synaptic tag." A later strong event triggers PRPs (Plasticity-Related Products) that sweep backward and consolidate tagged memories.

```
tag_memory(id, timestamp):
    tags.append({id, timestamp, expires_at = timestamp + tag_lifetime})

trigger_prp(event, importance):
    window_start = event.timestamp - backward_window  // default: 9 hours
    window_end = event.timestamp + forward_window     // default: 2 hours
    for each tag in tags:
        if tag.timestamp in [window_start, window_end]:
            consolidate(tag.memory_id, importance)
```

**For the daemon:** This is the computational model for belief revision. When a new elevated memory (high importance) arrives, it retroactively strengthens related memories in the temporal window. The daemon's "elevate" operation is a PRP trigger — it sweeps backward through the session timeline and boosts the stability of memories that preceded it.

### 66. Prediction Error Gating

When ingesting new content, the system compares against existing memories using embedding similarity:

| Similarity | Action | Meaning |
|------------|--------|---------|
| > 0.92 | REINFORCE existing | Almost identical — strengthen the existing memory |
| > 0.75 | UPDATE existing | Related — merge new information |
| < 0.75 | CREATE new | Novel — add as new memory |

**For the daemon:** This prevents duplicate memories before they enter the graph. When the daemon ingests a new memory, it checks embedding similarity against existing nodes. A near-duplicate is merged (updating the existing node's content and boosting its stability). A partial match updates the existing node with the new information. A novel memory creates a new node.

### 67. Reciprocal Rank Fusion for Hybrid Retrieval

Combines BM25 keyword search and semantic vector search into a single ranking:

```
RRF_score(d) = Σᵢ 1/(k + rank_i(d))
```

where `k = 60` (smoothing constant) and `rank_i(d)` is the rank of document `d` in ranking list `i`.

**For the daemon:** The daemon's retrieval combines exact match (tag/identifier lookup) with fuzzy match (embedding similarity). RRF fuses these into a single score without requiring the two ranking systems to be on the same scale.

---

## System Axioms (increasing strength)

| System | Condition on R | Axiom | Use in daemon |
|--------|---------------|-------|---------------|
| **K** | (none) | □(p→q) → (□p→□q) | Base: any graph edge is an accessibility relation |
| **D** | Serial (every world has at least one successor) | □p → ◇p | Every memory has at least one hop target (no dead ends) |
| **T** | Reflexive | □p → p | A memory is accessible from itself (self-loop for identity) |
| **B** | Symmetric | p → □◇p | Bidirectional hop_targets |
| **S4** | Reflexive + Transitive | □p → □□p | why_id chains (causal transitivity — if A caused B caused C, then A caused C) |
| **S5** | Equivalence relation | ◇p → □◇p | Elevated memory cluster (all elevated nodes are mutually accessible) |

---

## Package Structure

```
modal/
  types.go          # World, Frame, Model, Valuation, Formula
  frame.go          # Kripke frame construction, accessibility relations, path finding
  semantics.go      # Kripke semantics evaluator (truth at world w in model M)
  tableau.go        # Tableau-based satisfiability checker (tree of prefixed formulas)
  axioms.go         # System K, D, T, B, S4, S5 axiom schemas and frame conditions
  temporal.go       # Linear temporal logic (LTL) operators: □ (always), ◇ (eventually), U (until)
  epistemic.go      # Multi-agent knowledge operators: □_A, ◇_A, common knowledge
  fuzzy_bridge.go   # LE-FALC bridge: fuzzy truth values under modal operators
  parser.go         # Modal expression lexer/parser ("□(p → ◇q)", "K_A p", "◇(p U q)")
  system.go         # core.LogicSystem adapter
  modal_test.go     # comprehensive tests
```

---

## Phase 1: Foundation — Kripke Semantics

### `types.go`

```go
type World uint32  // VarID-style handle into symbol table

type Formula interface {
    Evaluate(w World, m *Model) (TruthValue, error)  // returns fuzzy-capable truth
}

// Atomic propositions
type Atom struct {
    ID   VarID
    Name string  // Pool-backed
}

// Modal operators
type Box struct { Formula }    // □ — necessity
type Diamond struct { Formula } // ◇ — possibility

// Classical connectives (reuse from classical/ package where possible)
type Not struct { Formula }
type And struct { Left, Right Formula }
type Or struct { Left, Right Formula }
type Implies struct { Antecedent, Consequent Formula }
type Iff struct { Left, Right Formula }
```

CC=1-3 per struct. All Formula types are value types (no boxing unless needed for the interface).

### `frame.go`

```go
type Frame struct {
    Worlds      []World            // Arena-backed
    Relations   map[RelationKey][]World  // source → []target
    // RelationKey = (source World, relationType uint8): why=0, how=1, hop=2, user-defined=3+
}

type Model struct {
    Frame      *Frame
    Valuation  map[VarID]map[World]TruthValue  // atom → world → truth value
}
```

CC=1-5 functions:
- `NewFrame(pool *memory.Pool) *Frame` — CC=1
- `AddWorld() World` — CC=1
- `AddRelation(source, target World, relType uint8)` — CC=2
- `Accessible(from World, relType uint8) []World` — CC=2, O(1) map lookup
- `IsAccessible(from, to World, relType uint8) bool` — CC=3, O(k) linear scan of edges
- `ReflexiveClosure(relType uint8)` — CC=4, O(W) add self-loops
- `TransitiveClosure(relType uint8)` — CC=5, O(W³) Floyd-Warshall
- `SymmetricClosure(relType uint8)` — CC=3, O(E) add reverse edges
- `FromDaemonGraph(whyIDs, howIDs, hopTargets map[uint64][]uint64) *Frame` — CC=4, O(V+E). Direct bridge from daemon's causal graph.

### `semantics.go`

```go
func (m *Model) Eval(formula Formula, world World) (TruthValue, error)
```

CC=6 dispatcher, delegates to per-operator evaluators (CC≤4 each):
- `evalAtom(w World) TruthValue` — O(1) lookup in Valuation map
- `evalBox(f Formula, w World, relType uint8) TruthValue` — O(A) where A = accessible worlds. Returns min truth across all accessible worlds. For crisp logic: true iff f is true in ALL accessible worlds.
- `evalDiamond(f Formula, w World, relType uint8) TruthValue` — O(A). Returns max truth across all accessible worlds. For crisp logic: true iff f is true in SOME accessible world.
- `evalNot`, `evalAnd`, `evalOr`, `evalImplies`, `evalIff` — O(1) each after recursive eval

Space: O(depth) for recursion stack. Max depth ≤ formula size.

---

## Phase 2: Tableau Prover

### `tableau.go`

Modal satisfiability checking via analytic tableaux. Given a formula φ, construct a tree where each branch is a set of prefixed formulas σ:ψ (read: "at world σ, ψ is true"). Apply expansion rules until either:
- All branches are closed (contradiction) → φ is unsatisfiable
- A branch is complete and open → φ is satisfiable, model can be extracted

```go
type TableauNode struct {
    Prefix     []World          // world path (Arena-backed)
    Formulas   []PrefixedFormula // (prefix, formula) pairs (Pool-backed)
    Children   []*TableauNode   // alternative branches
    Closed     bool
}

type PrefixedFormula struct {
    World   World
    Formula Formula
}
```

CC≤8 functions:
- `ProveSatisfiable(formula Formula, frame *Frame) (bool, *Model)` — CC=6. Returns satisfiable + counter-model if true.
- `ProveValid(formula Formula, frame *Frame) bool` — CC=2. ¬φ is unsatisfiable → φ is valid.
- `ProveEntails(premises []Formula, conclusion Formula, frame *Frame) bool` — CC=3. premises → conclusion valid?
- `expandBoxRule(node *TableauNode, pf PrefixedFormula)` — CC=4
- `expandDiamondRule(node *TableauNode, pf PrefixedFormula)` — CC=4
- `expandAndRule`, `expandOrRule`, `expandNotRule` — CC≤3 each
- `isContradictory(formulas []PrefixedFormula) bool` — CC=3

---

## Phase 3: Axiom Systems

### `axioms.go`

Pre-built frame transformers that enforce specific axiom system properties.

```go
func EnforceSystemK(frame *Frame)    // no-op, K holds in all frames
func EnforceSystemD(frame *Frame)    // add seriality: every world has ≥1 successor
func EnforceSystemT(frame *Frame)    // add reflexivity
func EnforceSystemB(frame *Frame)    // add symmetry
func EnforceSystemS4(frame *Frame)   // reflexivity + transitivity
func EnforceSystemS5(frame *Frame)   // equivalence relation
func ValidateFrameAgainst(frame *Frame, system System) error  // check if frame satisfies axioms
```

CC≤4 each. All O(V+E) or O(V³) for transitive closure.

```go
type System int
const (
    SystemK System = iota
    SystemD
    SystemT
    SystemB
    SystemS4
    SystemS5
)
```

---

## Phase 4: Temporal Logic (LTL)

### `temporal.go`

Linear temporal logic over session timelines. Each session turn is a world. The accessibility relation is time: R(s, t) iff t is the next turn after s.

```go
type TemporalModel struct {
    *Model
    Timeline []World  // ordered session worlds (Arena-backed)
}
```

Operators:
| Operator | Symbol | Meaning | Time complexity |
|----------|--------|---------|----------------|
| Always | □p | p holds now and in all future states | O(T) where T = timeline length from current world |
| Eventually | ◇p | p holds now or at some future state | O(T) |
| Next | ○p | p holds in the next state | O(1) |
| Until | p U q | p holds until q holds | O(T²) worst |
| Weak Until | p W q | p holds unless q holds (q may never hold) | O(T) |

CC≤8 functions:
- `NewTemporalModel(timeline []World, pool *memory.Pool) *TemporalModel` — CC=1
- `EvalAlways(p Formula, w World) TruthValue` — CC=3, O(T)
- `EvalEventually(p Formula, w World) TruthValue` — CC=3, O(T)
- `EvalNext(p Formula, w World) TruthValue` — CC=2, O(1)
- `EvalUntil(p, q Formula, w World) TruthValue` — CC=4, O(T²)
- `EvalWeakUntil(p, q Formula, w World) TruthValue` — CC=3, O(T)

**Daemon use:** Session timeline = temporal frame. "Was this fact true before the compaction event?" → □(before_compaction → fact). "Will this memory still be relevant in 10 turns?" → ◇(future ∧ memory).

---

## Phase 5: Epistemic Logic

### `epistemic.go`

Multi-agent knowledge and belief. Each agent has its own accessibility relation. □_A p means "agent A knows p."

```go
type EpistemicModel struct {
    *Model
    Agents      []AgentID            // Arena-backed
    Knowledge   map[AgentID]uint8    // agent → relation type for accessibility
    Belief      map[AgentID]uint8    // agent → relation type for belief (possibly different)
}

type AgentID uint32
```

CC≤6 functions:
- `NewEpistemicModel(agents []AgentID, pool *memory.Pool) *EpistemicModel` — CC=1
- `Knows(agent AgentID, p Formula, w World) TruthValue` — CC=2, delegates to evalBox with agent's relation
- `Believes(agent AgentID, p Formula, w World) TruthValue` — CC=2
- `CommonKnowledge(group []AgentID, p Formula, w World) TruthValue` — CC=5, O(A^d) where d = group size. Fixed-point over the transitive closure of the union of all group members' relations.
- `DistributedKnowledge(group []AgentID, p Formula, w World) TruthValue` — CC=4, O(A). Intersection of what each agent individually knows.
- `IsKnowledgeConsistent(agent AgentID, w World) bool` — CC=3. No world where □p and □¬p both hold for any p.

**Daemon use:** When the agent delegates a sub-task, the sub-agent's memory state is an epistemic world. □_main □_sub p means "the main agent knows that the sub-agent knows p." When sub-agent returns results, main checks `Believes(sub, result) ∧ ¬Knows(main, result)` → trusts but verifies.

---

## Phase 6: Fuzzy-Modal Bridge (LE-FALC)

### `fuzzy_bridge.go`

The LE-FALC research (2025) provides the formal foundation for fuzzy truth values under modal operators. This bridges `modal/` with `fuzzy/`.

```go
func BoxOverFuzzy(f Formula, w World, m *Model, tnorm func(a, b TruthValue) TruthValue) TruthValue
func DiamondOverFuzzy(f Formula, w World, m *Model, tconorm func(a, b TruthValue) TruthValue) TruthValue
```

For crisp frames (standard Kripke semantics):
- □p = min truth of p across all accessible worlds (Gödel t-norm)
- ◇p = max truth of p across all accessible worlds (Gödel t-conorm)

For weighted frames (daemon hop expansion, where edge weights ∈ [0,1]):
- □p = ⊗_{w' ∈ R(w)} (R(w,w') → p(w'))  — Łukasiewicz implication weighted by edge strength
- ◇p = ⊕_{w' ∈ R(w)} (R(w,w') ⊗ p(w'))  — Product t-norm weighted sum

CC≤6 functions:
- `BoxFuzzy(f Formula, w World, m *Model, cfg FuzzyConfig) TruthValue` — CC=5
- `DiamondFuzzy(f Formula, w World, m *Model, cfg FuzzyConfig) TruthValue` — CC=5
- `WeightedFrameAccessibility(from, to World, relType uint8, weights map[EdgeKey]float64) TruthValue` — CC=2
- `FuzzyEntailment(premises []Formula, conclusion Formula, frame *Frame, fuzzyCfg FuzzyConfig) TruthValue` — CC=6

**Daemon use:** The existing hop expansion weights (`etaWhy=0.7`, `etaHow=0.4`) become weighted accessibility relations. □_{why}(fact) = 0.7 means "fact is necessarily true via causal chain with confidence 0.7." Fuzzy modal operators propagate uncertainty through the graph.

---

## Phase 7: Parser & System Bridge

### `parser.go`

Lexer/parser for modal formulas. Shares `classical/lexer.go` Pool-backed token patterns.

Supported syntax:
```
□p           — necessity (ASCII: []p)
◇p           — possibility (ASCII: <>p)
□(p → ◇q)   — nested modal
□_A p        — epistemic: agent A knows p
◇(p U q)     — temporal until
□[why]p      — relation-qualified necessity (daemon: why_ids)
◇[how]p      — relation-qualified possibility (daemon: how_ids)
```

CC≤8 functions. Recursive descent parser. Precedence: □/◇/○ > ¬ > ∧ > ∨ > → > ↔.

### `system.go`

```go
type ModalLogicSystem struct {
    frame  *Frame
    model  *Model
    pool   *memory.Pool
}
```

CC=1-3 functions:
- `NewModalLogicSystem(pool *memory.Pool) *ModalLogicSystem` — CC=1
- `Name() string` — CC=1
- `Evaluate(expression string, ctx core.EvaluationContext) (bool, error)` — CC=4
- `Validate(expression string) error` — CC=3
- `SupportedOperators() []string` — CC=1

---

## Memory Allocation Strategy

| Structure | Allocator | Rationale |
|-----------|-----------|-----------|
| `Frame.Worlds` | `Arena` via `MustArenaSlice[World]` | Grow-only world list |
| `Frame.Relations` map | Go heap | Map keyed by `RelationKey` (uint64); bounded by edges |
| `Model.Valuation` map | Go heap | Map of maps; small per evaluation |
| `TableauNode` (struct) | `ShardedFreeList[TableauNode]` | Fixed ~64B struct; created per branch, freed on branch close. Thousands per `ProveSatisfiable()` |
| `PrefixedFormula` (struct) | `ShardedFreeList[PrefixedFormula]` | Fixed 16B struct; allocated per expansion step, freed on contradiction. Hot path. |
| Boxed `World` handles | `ShardedFreeList[World]` | 4B handle; allocated when world needs to escape stack (path tracking). Per-expansion. |
| `TableauNode.Prefix` | `Arena` via `MustArenaSlice[World]` | Per-branch world path — Arena because it grows as the branch deepens, freed once on branch close |
| `TableauNode.Formulas` | `Pool` via `MustPoolSlice[PrefixedFormula]` | Per-branch formula set — variable length, Reset() between branches |
| `TemporalModel.Timeline` | `Arena` via `MustArenaSlice[World]` | Ordered session worlds |
| Parser tokens | `Pool` via `MustPoolSlice[Token]` | Same as `classical/lexer.go` |
| Fuzzy-modal intermediates | `Pool` via `MustPoolSlice[TruthValue]` | Per-evaluation truth vectors |
| `EpistemicModel.Agents` | `Arena` via `MustArenaSlice[AgentID]` | Grow-only; agents added once |
| `Axiom closure results` | `Arena` via `MustArenaSlice[World]` | Reflexive/transitive/symmetric closure output — computed once, read many times |

> **Why ShardedFreeList for TableauNode and PrefixedFormula?** Tableau search explores multiple branches, some in parallel. Each expansion step creates a PrefixedFormula (world + formula pair). When a branch contradicts, the node and its formulas are freed. ShardedFreeList's per-worker shards eliminate lock contention across parallel branch workers. The 48-byte SIMD alignment ensures cache-line-friendly access patterns during the hot expansion loop.

---

## Function Complexity Budget

| File | Functions | Max CC |
|------|-----------|--------|
| `types.go` | 12 | 3 |
| `frame.go` | 10 | 5 |
| `semantics.go` | 10 | 6 |
| `tableau.go` | 12 | 8 |
| `axioms.go` | 7 | 4 |
| `temporal.go` | 8 | 8 |
| `epistemic.go` | 9 | 6 |
| `fuzzy_bridge.go` | 6 | 6 |
| `parser.go` | 14 | 8 |
| `system.go` | 5 | 4 |
| **Total** | **~93** | **8 (max)** |

---

## Implementation Order

### Phase 1: Kripke Semantics
1. **`types.go`** — World, Formula interface, Atom, Box, Diamond, Not, And, Or, Implies, Iff
2. **`frame.go`** — Frame, Model, accessibility relations, closures, daemon graph bridge
3. **`semantics.go`** — Kripke evaluator: truth at world w in model M
4. **Tests** — All above, 100% coverage

### Phase 2: Tableau Prover
5. **`tableau.go`** — Analytic tableau for satisfiability/validity/entailment
6. **Tests** — Known valid/invalid formulas, counter-model extraction

### Phase 3: Axiom Systems
7. **`axioms.go`** — System K/D/T/B/S4/S5 enforcers and validators
8. **Tests** — Frame property verification after closure operations

### Phase 4: Temporal Logic
9. **`temporal.go`** — LTL operators over session timelines
10. **Tests** — Temporal formula evaluation on known timelines

### Phase 5: Epistemic Logic
11. **`epistemic.go`** — Multi-agent knowledge, belief, common knowledge
12. **Tests** — Muddy children puzzle, delegation chains

### Phase 6: Fuzzy-Modal Bridge
13. **`fuzzy_bridge.go`** — Weighted accessibility, fuzzy entailment, LE-FALC integration
14. **Tests** — Fuzzy-modal formula on weighted frames, daemon hop weight validation

### Phase 7: Integration
15. **`parser.go`** — Modal formula lexer/parser
16. **`system.go`** — core.LogicSystem bridge
17. **Integration tests** — Daemon graph consistency checking, belief revision scenarios

---

---

## Verification Requirements (Every Phase)

### Automated
```bash
go test ./modal -v -cover          # 100% statement coverage
go test ./modal -race -timeout 60s # zero data races
go vet ./modal                     # no static analysis warnings
```

### Manual (per phase)
- [ ] Every function CC ≤ 10 (run `gocyclo` or manual count)
- [ ] Every exported symbol has a Go doc comment starting with the name
- [ ] Zero `make()` calls for slices (verify with `grep -r "make(\[\]" modal/`)
- [ ] Zero `new()` calls (verify with `grep -r "\bnew(" modal/`)
- [ ] `memory.Pool`/`memory.Arena` usage matches the allocation table for every structure
- [ ] `append()` only used on Pool/Arena-backed slices via `memory.ArenaAppend` or manual Pool expansion

### Per-Phase Gate
Each phase must pass all automated checks before the next phase begins. No exceptions.

---

## Research Targets (Deep Research Prompts)

### Target 1: Modal Logic for Knowledge Graph Verification (2025–2026)

Search for: "modal logic knowledge graph consistency verification 2025 2026", "Kripke semantics graph database", "description logic modal operators knowledge base", "weighted accessibility relation graph neural", "temporal description logic streaming data 2025".

What we need: How are modal operators (□, ◇) used to verify consistency in graph databases? Are there production systems using Kripke semantics over knowledge graphs?

### Target 2: Belief Revision Under Contradiction (2025–2026)

Search for: "belief revision modal logic 2025 2026", "AGM postulates fuzzy belief change", "paraconsistent modal logic dynamic", "iterated belief revision agent memory", "non-prioritized belief revision neural symbolic".

What we need: When a new elevated memory contradicts existing beliefs, what's the mathematically correct revision procedure? The AGM postulates are classic but fuzzy-modal extensions are new.

### Target 3: Epistemic Logic for Multi-Agent Delegation (2025–2026)

Search for: "epistemic logic multi-agent delegation 2025 2026", "dynamic epistemic logic knowledge update", "common knowledge distributed systems verification", "nested belief reasoning agent chains", "knowing whether vs knowing that agent planning".

What we need: How do production multi-agent systems model nested knowledge? What's the state of the art in epistemic planning?

### Target 4: Temporal Logic for Session Memory (2025–2026)

Search for: "linear temporal logic session memory 2025 2026", "LTL monitoring runtime verification", "temporal logic belief timeline agent", "past temporal operators memory retrieval", "metric temporal logic real-time agent".

What we need: How are temporal operators (always, eventually, until, since) used in agent memory systems? Is there work on "temporal relevance" — decaying belief strength over time using temporal modal operators?

### Target 5: Go Modal Logic Libraries (Landscape)

Search for: "Go modal logic library Kripke", "Go theorem prover modal", "Go description logic reasoner", "Go LTL model checker", "Go epistemic logic solver".

What we need: Does anything exist in Go? If not — same situation as fuzzy — we're first.

### Target 6: PINS-Based Incremental SAT for Modal Logic (LTSmin)

Search for: "partitioned next-state interface incremental SAT", "dependency matrix SAT solver", "FORCE variable ordering CDCL", "guard-based stubborn set partial order reduction modal logic".

What we need: Can PINS dependency matrices be applied to incremental SAT solving for modal logic? When a tableau step changes only one proposition, can we incrementally update the SAT encoding instead of rebuilding?

### Target 7: Symbolic Model Checking for Kripke Frames

Search for: "symbolic model checking Kripke frame", "BDD-based modal logic", "SDD sentential decision diagram modal", "vset abstraction model checking".

What we need: Can LTSmin's vset abstraction (BDD/SDD-backed world-set representation) be ported to Go? This would enable exponential compression of world-sets in the tableau prover.

### Target 8: TreeDBS-Style Compression for World Storage

Search for: "tree-based state compression model checking", "incremental hash tree state storage", "recursive state compression parallel".

What we need: LTSmin's TreeDBS incrementally updates only changed slots when storing a new state. Can we apply this to world storage in the Kripke frame, where most propositions are copied unchanged between accessible worlds?

---

## What's NOT in Scope (R4)

- Full first-order modal logic (FOL + modal) — stays propositional modal for R4
- Dynamic epistemic logic (action models, public announcements) — R5
- Probabilistic modal logic (probability distributions over worlds) — R5
- Deontic logic (obligation/permission) — separate library territory
- Model checking against external specifications — separate tool territory
