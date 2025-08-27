package logic

// Evaluator provides a fluent interface for chaining logical operations.
// It maintains an internal boolean state and allows operations to be
// chained together in a readable, method-chaining style.
//
// Example:
//
//	result := Eval(true).And(false).Or(true).Result() // true
type Evaluator struct {
	value bool
}

// Eval creates a new evaluator with the initial boolean value.
// This is the entry point for using the fluent evaluation interface.
//
// Example:
//
//	eval := Eval(true)
//	eval2 := Eval(false)
func Eval(initial bool) *Evaluator {
	return &Evaluator{value: initial}
}

// And performs logical AND between the current value and the given value.
// Updates the internal state and returns the evaluator for method chaining.
//
// Example:
//
//	result := Eval(true).And(false).Result() // false
func (e *Evaluator) And(other bool) *Evaluator {
	e.value = e.value && other
	return e
}

// Or performs logical OR between the current value and the given value.
// Updates the internal state and returns the evaluator for method chaining.
//
// Example:
//
//	result := Eval(false).Or(true).Result() // true
func (e *Evaluator) Or(other bool) *Evaluator {
	e.value = e.value || other
	return e
}

// Xor performs logical XOR between the current value and the given value.
// Updates the internal state and returns the evaluator for method chaining.
//
// Example:
//
//	result := Eval(true).Xor(false).Result() // true
//	result2 := Eval(true).Xor(true).Result() // false
func (e *Evaluator) Xor(other bool) *Evaluator {
	e.value = e.value != other
	return e
}

// Not performs logical NOT on the current value.
// Inverts the internal state and returns the evaluator for method chaining.
//
// Example:
//
//	result := Eval(true).Not().Result() // false
//	result2 := Eval(false).Not().Result() // true
func (e *Evaluator) Not() *Evaluator {
	e.value = !e.value
	return e
}

// Result returns the final boolean result of the evaluation chain.
// This should be called at the end of a method chain to get the computed value.
//
// Example:
//
//	result := Eval(true).And(false).Or(true).Not().Result() // false
func (e *Evaluator) Result() bool {
	return e.value
}
