# Research References

## License Statement

All code in this repository is original work. The repositories and papers listed
below were used for **algorithm reference and mathematical understanding only**.
No code was copied, translated, or adapted from any non-MIT source. Where MIT-
licensed reference implementations were consulted, algorithms were re-derived from
first principles and adapted to our off-heap memory model.

---

## Reference Implementations (MIT License)

### Kissat — SAT Solver
- **Repository**: https://github.com/arminbiere/kissat
- **Author**: Armin Biere (University of Freiburg)
- **License**: MIT
- **Reference scope**: Architecture reference. The CDCL solver structure (watched
  literals, 1st UIP analysis, VSIDS scoring, mode switching, reluctant doubling,
  WalkSAT pre-solving, Gaussian elimination for XOR, inprocessing pipeline) is
  derived from Kissat's architecture. Individual algorithms were re-implemented
  in Go with off-heap memory backing.
- **Key files referenced**:
  - `deduce.c` — two-watched-literal unit propagation
  - `analyze.c` — 1st UIP conflict analysis with clause minimization
  - `bump.c` — VSIDS activity scoring with EMA decay
  - `decide.c` — decision heuristics with heap and reluctant doubling
  - `mode.c` — focused/stable mode switching
  - `walk.c` — WalkSAT local search initialization
  - `heap.c` / `inlineheap.h` — binary max-heap for variable selection
  - `compact.c` / `eliminate.c` / `probe.c` / `subsume.c` — inprocessing

### Z3 — SMT Solver
- **Repository**: https://github.com/Z3Prover/z3
- **Organization**: Microsoft Research
- **License**: MIT
- **Reference scope**: DPLL(T) theory solver integration architecture. The
  `dpllt.go` module's TheoryPlugin interface and SAT→theory lemma feedback
  loop are derived from Z3's DPLL(T) pattern. No Z3 code was used.

### xDarkicex/memory — Off-Heap Allocator
- **Repository**: https://github.com/xDarkicex/memory
- **License**: MIT
- **Reference scope**: Production dependency. All solver memory is allocated
  through Pool (variable-size slices), Arena (grow-only trail), and ShardedFreeList
  (fixed-size Clause structs). The allocator is mmap-backed with lock-free CAS
  hot paths and Hyaline SMR for safe concurrent reclamation.

---

## Mathematical Reference (Papers)

The following academic papers provided the theoretical foundations. All algorithms
were re-derived from the mathematical descriptions in these papers.

### Core CDCL

- **GRASP: A Search Algorithm for Propositional Satisfiability**
  Silva & Sakallah. *IEEE Transactions on Computers*, 48(5), 1999.
  Foundation of conflict-driven clause learning.

- **Chaff: Engineering an Efficient SAT Solver**
  Moskewicz, Madigan, Zhao, Zhang, Malik. *DAC*, 2001.
  Two-watched-literal scheme and VSIDS decision heuristic.

- **Efficient Conflict Driven Learning in a Boolean Satisfiability Solver**
  Zhang, Madigan, Moskewicz, Malik. *ICCAD*, 2001.
  First UIP conflict analysis scheme.

### Clause Quality and Database Management

- **Predicting Learnt Clauses Quality in Modern SAT Solvers**
  Audemard & Simon. *IJCAI*, 2009.
  Literal Block Distance (LBD) metric and Glucose-style adaptive restarts.

- **On the Glucose SAT Solver**
  Audemard & Simon. *International Journal on Artificial Intelligence Tools*,
  2018.
  Glucose solver architecture and clause database management.

### Local Search

- **Noise Strategies for Improving Local Search**
  Selman, Kautz, Cohen. *AAAI*, 1994.
  WalkSAT algorithm — probabilistic greedy local search with random walk escape.

### Inprocessing

- **Inprocessing Rules**
  Järvisalo, Heule, Biere. *IJCAR*, 2012.
  Bounded variable elimination, subsumption, self-subsuming resolution, and
  failed literal probing during search.

### SAT Solver Architecture

- **CaDiCaL, Kissat, and the SAT Competition 2020–2024**
  Biere et al. *SAT Competition Reports*, 2020–2024.
  Modern SAT solver architecture and competition results.

### XOR Reasoning

- **Gaussian Elimination and XOR Constraints in SAT**
  Soos, Nohl, Castelluccia. *SAT*, 2009.
  Native XOR constraint handling via Gaussian elimination over GF(2).

### SMT and DPLL(T)

- **Z3: An Efficient SMT Solver**
  de Moura & Bjørner. *TACAS*, 2008.
  DPLL(T) architecture for theory solver integration with CDCL SAT core.

### Memory Management

- **Hyaline: Fast and Transparent Lock-Free Memory Reclamation**
  Nikolaev & Ravindran. *PLDI*, 2021.
  Hyaline SMR algorithm used by xDarkicex/memory's ShardedFreeList for safe
  concurrent memory reclamation.

- **A Scalable Lock-free Stack Algorithm**
  Hendler, Shavit, Yerushalmi. *SPAA*, 2004.
  Sharded Treiber stack design for concurrent free lists.

---

## Non-MIT Repositories — Mathematical Reference Only

The following repositories were consulted for mathematical understanding of
specific algorithms. **No code was used, translated, or adapted.**

### Spot — Automata Library
- **Repository**: https://gitlab.lrde.epita.fr/spot/spot
- **License**: GPLv3
- **Reference scope**: Automata-theoretic concepts only — emptiness checking via
  accepting SCC detection, formula simplification, BDD-based equivalence checking,
  and LTL splitting (obligation/suspendable/rest). These mathematical concepts
  were re-derived for the modal logic components of this project. The SAT solver
  package does not reference Spot.

### MiniSat — SAT Solver
- **Repository**: https://github.com/niklasso/minisat
- **Authors**: Een & Sörensson
- **License**: MIT
- **Reference scope**: Historical reference. MiniSat established the modern CDCL
  solver structure. Our solver derives from the more recent Kissat architecture
  but MiniSat's original design (watched literals, VSIDS, clause learning, Luby
  restarts) is foundational to all modern SAT solvers.

---

## Design Document

- **MODAL_PLAN.md** (this repository): Comprehensive architecture reference for
  the SAT solver and its integration with the modal logic pipeline. Specifies
  the off-heap memory constraints, cyclomatic complexity limits, and testing
  requirements that govern all implementation in this package.
