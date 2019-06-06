package command

// Handler must be implemented for a type to qualify as an command handler.
type Handler interface {
	Handle(cmd Command)
}
