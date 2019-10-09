package command

type ErrorInvalidCommand string

func (e ErrorInvalidCommand) Error() string {
	return string(e)
}

type ErrorCommandBusNotInitialized string

func (e ErrorCommandBusNotInitialized) Error() string {
	return string(e)
}

type ErrorCommandBusIsShuttingDown string

func (e ErrorCommandBusIsShuttingDown) Error() string {
	return string(e)
}

const (
	InvalidCommandError           = ErrorInvalidCommand("command: invalid command")
	CommandBusNotInitializedError = ErrorCommandBusNotInitialized("command: the command bus is not initialized")
	CommandBusIsShuttingDownError = ErrorCommandBusIsShuttingDown("command: the command bus is shutting down")
)
