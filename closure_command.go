package command

const ClosureIdentifier Identifier = "closure"

// Closure is the type used by the bus to handle closures
type Closure func() (data any, err error)

func (Closure) Identifier() Identifier {
	return ClosureIdentifier
}

type ClosureHandler struct {
}

func (hdl *ClosureHandler) Handles() Identifier {
	return ClosureIdentifier
}

func (hdl *ClosureHandler) Handle(cmd Command) (data any, err error) {
	if cmd, ok := any(cmd).(Closure); ok {
		return cmd()
	}
	return nil, InvalidClosureCommandError
}
