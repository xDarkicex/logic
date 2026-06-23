# Wire ExplainEngagement to CLI / Agent — Future Task

## Status

`Engine.ExplainEngagement(itemID string, numSamples int)` is implemented
and callable from Go. Not exposed to operators or agents yet.

## What it does

Runs a counterfactual BMS query: "what would downstream activation be
if memory itemID had NOT been retrieved?" Equations are auto-synthesized
from normalized causal graph edge weights. Returns expected value and
variance.

## Wiring needed

- **gRPC**: requires a new endpoint in `libravdb-contracts` (proto
  definition, server handler in daemon, client stub regeneration).
  Non-trivial — the contracts repo needs updating.

- **CLI**: once the gRPC endpoint exists, a `libravdbd explain <itemID>`
  command is ~30 lines following the pattern in `cmd/search.go`.

- **Agent**: once the gRPC endpoint exists, the agent calls it to
  debug retrieval quality ("was memory X causally important to this
  response?").

## Deferred

Not wired now — requires gRPC contract changes. The Go API is ready.
