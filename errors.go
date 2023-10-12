package command

// BusError is used to create errors originating from the command bus
type BusError string

// Error returns the string message of the error.
func (e BusError) Error() string {
	return string(e)
}

const (
	InvalidCommandError       = BusError("command: invalid command")
	BusNotInitializedError    = BusError("command: the bus is not initialized")
	BusIsShuttingDownError    = BusError("command: the bus is shutting down")
	OneHandlerPerCommandError = BusError("command: there can only be one handler per command")
	HandlerNotFoundError      = BusError("command: no handler found for the command provided")
)
