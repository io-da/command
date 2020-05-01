package command

// ErrorInvalidCommand is used when invalid commands are handled.
type ErrorInvalidCommand string

// Error returns the string message of ErrorInvalidCommand.
func (e ErrorInvalidCommand) Error() string {
	return string(e)
}

// ErrorBusNotInitialized is used when commands are handled but the bus is not initialized.
type ErrorBusNotInitialized string

// Error returns the string message of ErrorBusNotInitialized.
func (e ErrorBusNotInitialized) Error() string {
	return string(e)
}

// ErrorBusIsShuttingDown is used when commands are handled but the bus is shutting down.
type ErrorBusIsShuttingDown string

// Error returns the string message of ErrorBusIsShuttingDown.
func (e ErrorBusIsShuttingDown) Error() string {
	return string(e)
}

const (
	// InvalidCommandError is a constant equivalent of the ErrorInvalidCommand error.
	InvalidCommandError = ErrorInvalidCommand("command: invalid command")
	// BusNotInitializedError is a constant equivalent of the ErrorBusNotInitialized error.
	BusNotInitializedError = ErrorBusNotInitialized("command: the bus is not initialized")
	// BusIsShuttingDownError is a constant equivalent of the ErrorBusIsShuttingDown error.
	BusIsShuttingDownError = ErrorBusIsShuttingDown("command: the bus is shutting down")
)
