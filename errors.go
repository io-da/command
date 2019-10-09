package command

const (
	InvalidCommandError           = "command: invalid command"
	CommandBusNotInitializedError = "command: the command bus is not initialized"
	CommandBusIsShuttingDownError = "command: the command bus is shutting down"
)

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
