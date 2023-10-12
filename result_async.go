package command

import "sync/atomic"

// ResultAsync is the struct returned from async commands.
type ResultAsync struct {
	*Result
	handled *uint32
	pending chan bool
	err     error
}

func newResultAsync() *ResultAsync {
	return &ResultAsync{
		Result:  newResult(),
		handled: new(uint32),
		pending: make(chan bool, 1),
	}
}

//------Fetch Data------//

// Await for the data from the return of the command
func (res *ResultAsync) Await() error {
	if !res.isHandled() {
		<-res.pending
		res.setHandled()
	}
	return res.err
}

// Get retrieves the data from the return of the first command.
func (res *ResultAsync) Get() (any, error) {
	if err := res.Await(); err != nil {
		return nil, err
	}
	return res.Result.Get(), nil
}

//------Internal------//

func (res *ResultAsync) done() {
	res.pending <- true
}

func (res *ResultAsync) setHandled() {
	atomic.CompareAndSwapUint32(res.handled, 0, 1)
}

func (res *ResultAsync) isHandled() bool {
	return atomic.LoadUint32(res.handled) == 1
}

func (res *ResultAsync) fail(err error) {
	res.err = err
	res.done()
}
