package fuzzy

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/xDarkicex/memory"
)

// InferenceEngine defines the common interface for Mamdani and TSK engines
// so they can be managed by the Arbiter.
type InferenceEngine interface {
	// For arbiter, we'll evaluate and then defuzzify to a single float64 to compare.
	// Mamdani returns a *FuzzySet which we will defuzzify via a wrapper if needed,
	// but Arbiter needs a consistent return. Let's assume engines return interface{}
	// or we provide a wrapper. The plan states Mamdani returns *FuzzySet and TSK returns float64.
	// We'll use a generic Evaluate wrapper interface.
}

// Strategy wrapper for Arbiter
type Strategy interface {
	Evaluate(inputs map[VarID]float64, pool *memory.Pool) (float64, error)
}

// MamdaniStrategy wraps MamdaniEngine to return a defuzzified value.
type MamdaniStrategy struct {
	Engine     *MamdaniEngine
	DefuzzFunc func(*FuzzySet) float64
}

// Evaluate runs Mamdani inference and defuzzifies the result.
func (s *MamdaniStrategy) Evaluate(inputs map[VarID]float64, pool *memory.Pool) (float64, error) {
	fs, err := s.Engine.Evaluate(inputs, pool)
	if err != nil {
		return 0, err
	}
	return s.DefuzzFunc(fs), nil
}

// TSKStrategy wraps TSKEngine.
type TSKStrategy struct {
	Engine *TSKEngine
}

// Evaluate runs TSK inference.
func (s *TSKStrategy) Evaluate(inputs map[VarID]float64, pool *memory.Pool) (float64, error) {
	// TSK doesn't need pool for its current implementation but fulfills interface
	return s.Engine.Evaluate(inputs)
}

// strategyRecord holds a strategy and its atomic reliability score.
type strategyRecord struct {
	name        string
	strategy    Strategy
	reliability uint64 // store as bits of float64 for atomic access
}

// Arbiter manages multiple strategies and evaluates them concurrently,
// picking the result from the highest-scoring strategy that successfully fires.
type Arbiter struct {
	strategies []*strategyRecord
	mu         sync.RWMutex // For adding strategies, though usually done at setup
}

// NewArbiter creates a new Arbiter.
func NewArbiter() *Arbiter {
	return &Arbiter{}
}

// AddStrategy registers a new strategy with an initial score.
func (a *Arbiter) AddStrategy(name string, strategy Strategy, initialScore float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.strategies = append(a.strategies, &strategyRecord{
		name:        name,
		strategy:    strategy,
		reliability: math.Float64bits(initialScore),
	})
}

// UpdateReliability atomically adjusts the score of a strategy.
func (a *Arbiter) UpdateReliability(name string, delta float64) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, rec := range a.strategies {
		if rec.name == name {
			for {
				oldBits := atomic.LoadUint64(&rec.reliability)
				oldVal := math.Float64frombits(oldBits)
				newVal := oldVal + delta
				newBits := math.Float64bits(newVal)
				if atomic.CompareAndSwapUint64(&rec.reliability, oldBits, newBits) {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("strategy not found: %s", name)
}

// arbiterResult holds the output from one strategy evaluation.
type arbiterResult struct {
	name  string
	score float64
	value float64
	err   error
}

// Select runs all strategies concurrently and returns the best valid result.
// Split into dispatchEngines and collectResults to keep CC <= 10.
func (a *Arbiter) Select(inputs map[VarID]float64, pool *memory.Pool) (float64, string, error) {
	a.mu.RLock()
	strats := a.strategies
	a.mu.RUnlock()

	if len(strats) == 0 {
		return 0, "", fmt.Errorf("no strategies registered")
	}

	resultsCh := make(chan arbiterResult, len(strats))
	a.dispatchEngines(strats, inputs, pool, resultsCh)
	return a.collectResults(len(strats), resultsCh)
}

// dispatchEngines launches goroutines for each strategy. CC=3.
func (a *Arbiter) dispatchEngines(strats []*strategyRecord, inputs map[VarID]float64, pool *memory.Pool, resultsCh chan<- arbiterResult) {
	var wg sync.WaitGroup
	wg.Add(len(strats))

	for _, rec := range strats {
		go func(r *strategyRecord) {
			defer wg.Done()
			val, err := r.strategy.Evaluate(inputs, pool)
			score := math.Float64frombits(atomic.LoadUint64(&r.reliability))
			resultsCh <- arbiterResult{
				name:  r.name,
				score: score,
				value: val,
				err:   err,
			}
		}(rec)
	}

	// Close channel when all are done
	go func() {
		wg.Wait()
		close(resultsCh)
	}()
}

// collectResults gathers results and selects the one with the highest score. CC=6.
func (a *Arbiter) collectResults(count int, resultsCh <-chan arbiterResult) (float64, string, error) {
	var bestResult arbiterResult
	found := false

	for res := range resultsCh {
		if res.err != nil {
			continue // Skip failed evaluations
		}
		if !found || res.score > bestResult.score {
			bestResult = res
			found = true
		}
	}

	if !found {
		return 0, "", fmt.Errorf("all strategies failed")
	}
	return bestResult.value, bestResult.name, nil
}
