package command

type processTypes interface {
	*Result | *ResultAsync
}

type process[T processTypes] struct {
	cmd Command
	res T
}

func newProcess[T processTypes](cmd Command, res T) *process[T] {
	return &process[T]{
		cmd: cmd,
		res: res,
	}
}
