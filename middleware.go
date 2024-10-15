package command

// Next represents the type used to process the next step in the middlewares pipeline.
type Next func(cmd Command) (any, error)

// Middleware must be implemented for a type to qualify as a command middleware.
// Middlewares process commands before being provided to their respective command handler.
// Middlewares may also execute logic after the command execution.
type Middleware interface {
	Handle(cmd Command, next Next) (any, error)
}
