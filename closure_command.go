package command

// ClosureCommand is the type used by the bus to handle closures
type ClosureCommand func() (data any, err error)
