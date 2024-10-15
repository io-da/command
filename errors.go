package command

// BusError is used to create errors originating from the command bus
type BusError string

// Error returns the string message of the error.
func (e BusError) Error() string {
	return string(e)
}

const (
	// InvalidCommandError will be returned when attempting to handle an invalid command.
	InvalidCommandError = BusError("command: invalid command")
	// BusNotInitializedError will be returned when attempting to handle a command before the bus is initialized.
	BusNotInitializedError = BusError("command: the bus is not initialized")
	// BusIsShuttingDownError will be returned when attempting to handle a command while the bus is shutting down.
	BusIsShuttingDownError = BusError("command: the bus is shutting down")
	// OneHandlerPerCommandError will be returned when attempting to initialize the bus with more than one handler listening to the same command.
	OneHandlerPerCommandError = BusError("command: there can only be one handler per command")
	// HandlerNotFoundError will be returned when no handler is found to the provided command.
	HandlerNotFoundError = BusError("command: no handler found for the command provided")
	// EmptyAwaitListError will be returned when attempting to await an empty AwaitList
	EmptyAwaitListError = BusError("command: await list is empty")
	// InvalidClosureCommandError will be returned when attempting to handle a command with the closure identifier but invalid type
	InvalidClosureCommandError = BusError("command: invalid closure command")
)
