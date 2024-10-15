package command

// Identifier is used to create a consistent identity solution for commands
type Identifier string

// Command is the interface that must be implemented by any type to be considered a command.
type Command interface {
	Identifier() Identifier
}
