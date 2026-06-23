# Cognitive Memory Models for libravdbd

Mathematical models extracted from Vestige (AGPL-3.0, math only) applicable to the daemon's memory retrieval, compaction, and recall scoring.

## 1. FSRS-6 Power Law Forgetting

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

**For the daemon:** Every memory node in the graph has a `stability` and `difficulty` field. The retrievability `R(t,S)` is the probability that the memory is recallable at time `t`. When the daemon retrieves a memory, it updates the FSRS state based on whether the memory was actually useful (the user's feedback = the "grade"). Memories that are consistently useful get higher stability and are scheduled further apart. This replaces ad-hoc recency scoring.

## 2. Dual-Strength Memory Model

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

## 3. Spreading Activation Network

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

**For the daemon:** The daemon's `hop_targets` ARE the spreading activation edges. When a memory is recalled, activation spreads along `hop_targets` to related memories. The decay factor (0.7 per hop) determines how far activation reaches. This is mathematically equivalent to fuzzy ◇ evaluation: `◇P` at world `w` computes max truth of `P` across all `R(w)`-accessible worlds, weighted by edge strength. Make the decay factor explicit in the config rather than implicit.

## 4. Synaptic Tagging & Capture (Retroactive Importance)

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

## 5. Prediction Error Gating

When ingesting new content, the system compares against existing memories using embedding similarity:

| Similarity | Action | Meaning |
|------------|--------|---------|
| > 0.92 | REINFORCE existing | Almost identical — strengthen the existing memory |
| > 0.75 | UPDATE existing | Related — merge new information |
| < 0.75 | CREATE new | Novel — add as new memory |

**For the daemon:** This prevents duplicate memories before they enter the graph. When the daemon ingests a new memory, it checks embedding similarity against existing nodes. A near-duplicate is merged (updating the existing node's content and boosting its stability). A partial match updates the existing node with the new information. A novel memory creates a new node.

## 6. Reciprocal Rank Fusion for Hybrid Retrieval

Combines BM25 keyword search and semantic vector search into a single ranking:

```
RRF_score(d) = Σᵢ 1/(k + rank_i(d))
```

where `k = 60` (smoothing constant) and `rank_i(d)` is the rank of document `d` in ranking list `i`.

**For the daemon:** The daemon's retrieval combines exact match (tag/identifier lookup) with fuzzy match (embedding similarity). RRF fuses these into a single score without requiring the two ranking systems to be on the same scale.

## Daemon-Vestige Gap Analysis

| What Vestige has | Daemon status | Gap |
|-----------------|---------------|-----|
| FSRS-6 power law forgetting | Anchor-based compaction | Daemon uses heuristic decay — FSRS-6 is mathematically optimal from 700M+ reviews |
| Dual-strength (storage ≠ retrieval) | Single timestamp-based recency | No distinction between "well stored" and "easily retrieved" |
| Spreading activation (0.7 decay/hop) | hop_targets expansion | Same concept! Make 0.7 decay explicit in config |
| Synaptic tagging (retroactive 9hr window) | Elevate operation | Elevate should retroactively boost memories in temporal window |
| Prediction error gating (0.92/0.75 thresholds) | Similarity-based dedup | Same concept, formalize thresholds |
| RRF hybrid ranking fusion | BM25 + semantic search | Likely already implemented |
