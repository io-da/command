package command

// ErrorHandler must be implemented for a type to qualify as an error handler.
type ErrorHandler interface {
	Handle(cmd Command, err error)
}
