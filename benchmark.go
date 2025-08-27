package logic

import "time"

// Operation represents a benchmarkable operation with a name and function.
// It's used by the Benchmark type to organize and execute performance tests.
type Operation struct {
	// Name is a descriptive name for the operation being benchmarked
	Name string

	// Func is the operation to be executed and measured
	Func func() bool
}

// Benchmark provides utilities for performance testing of boolean operations.
// It allows you to add multiple operations and execute them to compare
// their performance characteristics.
//
// Example:
//
//	benchmark := NewBenchmark()
//	benchmark.Add("AND operation", func() bool { return And(true, false) })
//	benchmark.Add("OR operation", func() bool { return Or(true, false) })
//	benchmark.Run()
type Benchmark struct {
	operations []Operation

	// Results stores the boolean results of each operation after execution
	Results []bool
}

// NewBenchmark creates a new benchmark instance.
// The benchmark starts empty and operations can be added using the Add method.
//
// Example:
//
//	benchmark := NewBenchmark()
func NewBenchmark() *Benchmark {
	return &Benchmark{
		operations: make([]Operation, 0),
		Results:    make([]bool, 0),
	}
}

// Add adds an operation to the benchmark.
// The operation will be executed when Run() is called.
// The name should be descriptive to help identify the operation in results.
//
// Example:
//
//	benchmark.Add("Complex expression", func() bool {
//		return And(Or(true, false), Xor(true, false))
//	})
func (b *Benchmark) Add(name string, fn func() bool) {
	b.operations = append(b.operations, Operation{
		Name: name,
		Func: fn,
	})
}

// Run executes all benchmark operations that have been added.
// It stores the results in the Results slice, which can be accessed
// after execution for verification or analysis.
//
// Note: This is a simple execution framework. For detailed performance
// measurements, use Go's built-in testing.B benchmark framework.
//
// Example:
//
//	benchmark.Run()
//	for i, result := range benchmark.Results {
//		fmt.Printf("Operation %d result: %v\n", i, result)
//	}
func (b *Benchmark) Run() {
	b.Results = make([]bool, len(b.operations))

	for i, op := range b.operations {
		start := time.Now()
		result := op.Func()
		_ = time.Since(start) // Duration could be stored if needed
		b.Results[i] = result
	}
}
