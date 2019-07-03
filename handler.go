package command

// Handler must be implemented for a type to qualify as a command handler.
type Handler interface {
	Handle(cmd Command) error
}
