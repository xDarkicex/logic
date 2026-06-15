package modal

import (
	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// AgentID identifies an agent in a multi-agent epistemic model.
type AgentID uint32

// agentRel maps an agent to a relation type for knowledge or belief.
type agentRel struct {
	agent AgentID
	rel   RelationType
}

// EpistemicModel extends a Kripke model with multiple agents,
// each with their own accessibility relation for knowledge and belief.
// Agent mappings are Pool-backed slices (no maps — linear scan, agent count is small).
type EpistemicModel struct {
	*Model
	agents    []AgentID   // Arena-backed
	knowledge []agentRel  // Pool-backed, agent → relation for knowledge
	belief    []agentRel  // Pool-backed, agent → relation for belief
	pool      *memory.Pool
}

// NewEpistemicModel creates an epistemic model with the given agents.
// Agents are copied into an Arena-backed slice; relation mappings are Pool-backed.
// CC=1.
func NewEpistemicModel(frame *Frame, agents []AgentID, numVars int, pool *memory.Pool, arena *memory.Arena) *EpistemicModel {
	ag := memory.MustArenaSlice[AgentID](arena, len(agents))
	ag = ag[:len(agents)]
	copy(ag, agents)

	k := memory.MustPoolSlice[agentRel](pool, len(agents))
	k = k[:0]
	b := memory.MustPoolSlice[agentRel](pool, len(agents))
	b = b[:0]

	return &EpistemicModel{
		Model:     NewModel(frame, numVars, pool, arena),
		agents:    ag,
		knowledge: k,
		belief:    b,
		pool:      pool,
	}
}

// SetKnows assigns a relation type for an agent's knowledge accessibility.
func (em *EpistemicModel) SetKnows(agent AgentID, rel RelationType) {
	for i := range em.knowledge {
		if em.knowledge[i].agent == agent {
			em.knowledge[i].rel = rel
			return
		}
	}
	em.knowledge = append(em.knowledge, agentRel{agent: agent, rel: rel})
}

// SetBelief assigns a relation type for an agent's belief accessibility.
func (em *EpistemicModel) SetBelief(agent AgentID, rel RelationType) {
	for i := range em.belief {
		if em.belief[i].agent == agent {
			em.belief[i].rel = rel
			return
		}
	}
	em.belief = append(em.belief, agentRel{agent: agent, rel: rel})
}

// knowRel returns the knowledge relation for an agent, or RelCausal if unset.
func (em *EpistemicModel) knowRel(agent AgentID) RelationType {
	for _, entry := range em.knowledge {
		if entry.agent == agent {
			return entry.rel
		}
	}
	return RelCausal
}

// beliefRel returns the belief relation for an agent, or RelProcedural if unset.
func (em *EpistemicModel) beliefRel(agent AgentID) RelationType {
	for _, entry := range em.belief {
		if entry.agent == agent {
			return entry.rel
		}
	}
	return RelProcedural
}

// Agents returns the list of agents.
func (em *EpistemicModel) Agents() []AgentID { return em.agents }

// Knows evaluates □_agent p: agent knows p, meaning p holds in all worlds
// accessible via the agent's knowledge relation. Delegates to Box evaluation.
// CC=2.
func (em *EpistemicModel) Knows(agent AgentID, p Formula, w World) (TruthValue, error) {
	rel := em.knowRel(agent)
	b := Box{Formula: p, Rel: rel}
	return b.Evaluate(w, em.Model)
}

// Believes evaluates B_agent p: agent believes p, similar to Knows but uses
// the belief accessibility relation (which may differ from knowledge).
// CC=2.
func (em *EpistemicModel) Believes(agent AgentID, p Formula, w World) (TruthValue, error) {
	rel := em.beliefRel(agent)
	b := Box{Formula: p, Rel: rel}
	return b.Evaluate(w, em.Model)
}

// CommonKnowledge evaluates C_G p at world w: everyone in group G knows p,
// everyone knows everyone knows p, and so on ad infinitum.
// Computed as the greatest fixpoint of CK(p) = p ∧ □_all CK(p).
// O(W²·|G|·E) worst case. CC=5.
func (em *EpistemicModel) CommonKnowledge(group []AgentID, p Formula, w World) (TruthValue, error) {
	wc := em.Frame().WorldCount()
	if wc == 0 {
		return 0.0, nil
	}
	// Start with all worlds where p is true
	ck := memory.MustPoolSlice[bool](em.pool, wc)
	ck = ck[:wc]
	for i := World(0); i < World(wc); i++ {
		tv, err := p.Evaluate(i, em.Model)
		if err != nil {
			return 0, err
		}
		ck[i] = tv > 0
	}

	// Iteratively remove worlds that fail the □_all condition
	changed := true
	for changed {
		changed = false
		for i := World(0); i < World(wc); i++ {
			if !ck[i] {
				continue
			}
			if !em.allKnow(ck, group, i) {
				ck[i] = false
				changed = true
			}
		}
	}

	if ck[w] {
		return 1.0, nil
	}
	return 0.0, nil
}

// allKnow checks whether every world accessible from w via any group agent
// remains in the common knowledge set.
// CC=3.
func (em *EpistemicModel) allKnow(ck []bool, group []AgentID, w World) bool {
	wc := em.Frame().WorldCount()
	for _, agent := range group {
		rel := em.knowRel(agent)
		for v := World(0); v < World(wc); v++ {
			if em.Frame().IsAccessible(w, v, rel) && !ck[v] {
				return false
			}
		}
	}
	return true
}

// DistributedKnowledge evaluates D_G p at world w: the group G can deduce p
// by pooling their knowledge. p must hold in all worlds accessible via EVERY
// agent's relation simultaneously (the intersection of accessibility relations).
// CC=4.
func (em *EpistemicModel) DistributedKnowledge(group []AgentID, p Formula, w World) (TruthValue, error) {
	wc := em.Frame().WorldCount()
	if wc == 0 || len(group) == 0 {
		return 0.0, nil
	}

	// Build the intersection: worlds that are accessible by ALL group agents
	accessible := memory.MustPoolSlice[bool](em.pool, wc)
	accessible = accessible[:wc]
	for i := World(0); i < World(wc); i++ {
		accessible[i] = true
	}

	for _, agent := range group {
		rel := em.knowRel(agent)
		for v := World(0); v < World(wc); v++ {
			if !em.Frame().IsAccessible(w, v, rel) {
				accessible[v] = false
			}
		}
	}

	// Check p holds in all worlds in the intersection
	result := TruthValue(1.0)
	for v := World(0); v < World(wc); v++ {
		if !accessible[v] {
			continue
		}
		tv, err := p.Evaluate(v, em.Model)
		if err != nil {
			return 0, err
		}
		if tv < result {
			result = tv
		}
		if result == 0 {
			return 0.0, nil
		}
	}
	return result, nil
}

// IsKnowledgeConsistent checks that agent has no contradictory knowledge:
// there is no world with accessible successors where the agent simultaneously
// knows p and ¬p for any atomic p.
// Vacuously true boxes (no accessible worlds) are skipped — they represent
// "knows nothing" rather than "knows a contradiction".
// CC=4.
func (em *EpistemicModel) IsKnowledgeConsistent(agent AgentID) bool {
	rel := em.knowRel(agent)
	wc := em.Frame().WorldCount()
	numAtoms := em.NumVars()

	for atom := 0; atom < numAtoms; atom++ {
		a := Atom{ID: fuzzy.VarID(atom)}
		na := Not{Formula: a}
		for w := World(0); w < World(wc); w++ {
			if len(em.Frame().Accessible(w, rel)) == 0 {
				continue // vacuous knowledge — skip
			}
			tvP, _ := Box{Formula: a, Rel: rel}.Evaluate(w, em.Model)
			tvN, _ := Box{Formula: na, Rel: rel}.Evaluate(w, em.Model)
			if tvP > 0 && tvN > 0 {
				return false
			}
		}
	}
	return true
}
