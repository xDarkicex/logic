# CLAUDE.md — Hard Constraints for This Project

## NEVER BREAK THESE

### 1. Memory: xDarkicex/memory ONLY

Every allocation MUST use `github.com/xDarkicex/memory`. The Go GC must never scan
reasoning data.

| Allocator | Use for | Notes |
|-----------|---------|-------|
| `memory.Pool` via `MustPoolSlice[T]` | Variable-size slices. Bulk `Reset()` between evaluations | Default choice for slice backing |
| `memory.Arena` via `MustArenaSlice[T]` + `ArenaAppend` | Grow-only data. `Free()` on teardown | Trail entries, tableau branches |
| `memory.ShardedFreeList` via `Acquire()` / `Release()` | Fixed-size structs with high churn. FASTEST at 25.8× vs `make()` | NOT always correct — only for compile-time-known sizes. Use Pool for variable-size |

**FORBIDDEN:**
- `make([]T, ...)` — use `memory.MustPoolSlice[T](pool, n)` or `memory.MustArenaSlice[T](arena, n)`
- `new(T)` or `&T{}` for any slice-backed struct — allocate backing arrays from Pool/Arena
- `append()` on a non-Pool/non-Arena slice — use `ArenaAppend` for Arena slices, or ensure Pool backing with sufficient capacity
- `make(map[K]V)` — allowed ONLY for small lookup maps (≤ 50 entries, bounded). All backing slices within maps must be Pool/Arena

Every file that allocates must import `"github.com/xDarkicex/memory"`.

### 2. Cyclomatic Complexity ≤ 10

Every function must have CC ≤ 10. Hot-path functions (evaluators, graph traversal,
propagation) target CC ≤ 6. Extract helpers aggressively.

Verify with: `gocyclo -over 10 .` or manual counting (1 + if/for/case/&&/|| count).

### 3. Testing: Every Function, Race Clean

- Every exported function must have at least one dedicated test case
- Every internal helper must be exercised through exported function tests
- `go test -race ./...` must pass on every commit
- Probe tests (same-package `_test.go` files) are encouraged for hard-to-reach paths

### 4. Go Doc Comments

Every exported type, function, method, and constant must have a Go doc comment
starting with the name:

```go
// CausalGraph is a directed mixed graph with Pool-backed adjacency lists.
type CausalGraph struct { ... }
```

## Tools Available

- `go test -race ./...` — race detector
- `go test -cover ./...` — coverage
- `gocyclo -over 10 .` — complexity check
- `staticcheck ./...` — static analysis (installed on this machine)
- `gofmt -w .` — format before commit

## Planning Docs

**CAUSAL_PLAN.md** — located at repo root (`./CAUSAL_PLAN.md`). This is the
authoritative design document for the causal logic package. You MUST reference it
before and during any implementation in `causal/`. It specifies the SOTA algorithms,
Go API designs, data structures, implementation phases, mathematical foundations,
and performance patterns. Do not deviate from the plan without explicit instruction.

Planning documents (CAUSAL_PLAN.md, MODAL_PLAN.md) are internal only. They do not
go on git. Final documentation goes in `docs/`.

## Branch Strategy

- `main` — stable, merge from feature branches
- `R4`, `R5`, etc. — release/feature branches
- Commit messages follow repo convention: `pkg: brief description`
