package command

// Command is the interface that must be implemented by any type to be considered a command.
type Command interface {
	ID() []byte
}
