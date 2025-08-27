package logic

import "fmt"

// LogicError represents an error that occurred during a logic operation.
// It provides context about which operation failed and why, making
// debugging easier for users of the package.
type LogicError struct {
	// Op is the name of the operation that caused the error
	Op string

	// Message provides additional details about what went wrong
	Message string
}

// Error implements the error interface for LogicError.
// It returns a formatted error message that includes both the operation
// name and the specific error details.
//
// Example output: "logic operation 'BoolVector.And': vector length mismatch"
func (e *LogicError) Error() string {
	return fmt.Sprintf("logic operation '%s': %s", e.Op, e.Message)
}

// NewLogicError creates a new LogicError with the specified operation and message.
// This is the preferred way to create logic errors within the package.
//
// Example:
//
//	err := NewLogicError("BoolVector.And", "vector length mismatch")
//	return nil, err
func NewLogicError(operation, message string) *LogicError {
	return &LogicError{
		Op:      operation,
		Message: message,
	}
}
