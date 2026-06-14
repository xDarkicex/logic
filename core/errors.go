package core

import (
	"fmt"
)

// LogicError represents an error in logic operations
type LogicError struct {
	System   string
	Op       string
	Message  string
	Position int
}

func (e *LogicError) Error() string {
	if e.System != "" {
		return fmt.Sprintf("logic error in %s.%s: %s", e.System, e.Op, e.Message)
	}
	return fmt.Sprintf("logic error in %s: %s", e.Op, e.Message)
}

func NewLogicError(system, operation, message string) *LogicError {
	return &LogicError{
		System:  system,
		Op:      operation,
		Message: message,
	}
}

// Backwards compatibility
func NewError(operation, message string) *LogicError {
	return NewLogicError("", operation, message)
}

// ContradictionError represents an error where contradictory beliefs were encountered.
// Instead of a hard failure, it carries the degraded result so the system can continue operating.
type ContradictionError struct {
	*LogicError
	DegradedResult interface{}
}

func NewContradictionError(system, operation, message string, degradedResult interface{}) *ContradictionError {
	return &ContradictionError{
		LogicError:     NewLogicError(system, operation, message),
		DegradedResult: degradedResult,
	}
}

func (e *ContradictionError) Error() string {
	return fmt.Sprintf("%s (Degraded Result Provided)", e.LogicError.Error())
}
