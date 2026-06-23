# Actual Cause Response Attribution

## Status

Daemon wired. `Engine.AttributeCause(candidateIDs, claimID)` and
`TopoRegistry.AttributeCause(candidateIDs, claimID)` are implemented.
SVAR equations auto-synthesized via `buildEquationsLocked` (shared with
`CounterfactualQuery`). Not exposed via CLI/gRPC — same contracts gap
as `ExplainEngagement`. Claim extraction is a model-side task, not
implemented.

## Problem

When a user asks "why did you say that?", the system should attribute the
response to the specific memories that *actually caused* it, not just
correlated or semantically similar ones.

`MinimalCausalExplanation` (already wired in `TopoRegistry`) answers
"which ancestors are sufficient to explain this?" — a covering set.
`ActualCause` answers a different question: "did memory X *cause* the
model to say Y under the modified HP definition?"

## How it works

The modified HP definition (Halpern & Pearl) says X=x is an actual cause
of Y=y if there exists a contingency set W such that:

1. In the actual world: X=x, W=w, Y=y
2. In the counterfactual: X=x', W=w, Y≠y

The SAT solver searches over contingency set sizes k=0..N-2. If a
satisfying assignment exists, X is an actual cause. `Responsibility`
returns 1/(1+|W|) — the smaller the contingency set needed, the higher
the responsibility.

## User-facing flow

```
User: "why did you say X?"
  → Agent extracts claim X from its last response
  → For each retrieved memory M_i in the context window:
      ActualCause(M_i, value=1.0, claim_node, value=1.0, equations)
  → Returns ranked list: "based on memory Y (responsibility 0.50),
    memory Z (responsibility 0.33), ..."
```

The equations are the same SVAR synthesis already built for
`ExplainEngagement` (`CounterfactualQuery` in `TopoRegistry`).

## Requirements

1. **Claim extraction**: the agent must identify the *specific claim*
   in its response that the user is questioning. This is a model-side
   task, not a daemon task. **Deferred.**

2. **API surface**: `TopoRegistry.AttributeCause(candidateIDs, claimID)`
   and `Engine.AttributeCause(candidateIDs, claimID)`. **Done.**

3. **CLI / Agent endpoint**: same gRPC gap as `ExplainEngagement`.
   **Deferred** until contracts work is scheduled.

4. **Equation synthesis**: `buildEquationsLocked` extracted from
   `CounterfactualQuery`, shared by both methods. **Done.**

## Dependency

- `cognitive-engine/causal/actual_causality.go` — `ActualCause`, `Responsibility`,
  `MinimalContingencySet` (fully implemented, tested)
- `logic/sat` — CDCL SAT solver (already wired via `sat_bridge.go`)
- `TopoRegistry.CounterfactualQuery` — equation synthesis pattern to follow
