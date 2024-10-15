package command

type AsyncResult struct {
	Index int
	Data  any
	Err   error
}

func (res AsyncResult) Get() (any, error) {
	return res.Data, res.Err
}
