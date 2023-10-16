package command

// InwardMiddleware must be implemented for a type to qualify as an inward command middleware.
// An inward middleware process a command before being provided to the respective command handler.
type InwardMiddleware interface {
	HandleInward(cmd Command) error
}

// OutwardMiddleware must be implemented for a type to qualify as an outward command middleware.
// An outward middleware process the command after being provided to the respective command handler.
type OutwardMiddleware interface {
	HandleOutward(cmd Command, data any, err error) (any, error)
}
