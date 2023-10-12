package command

import "sync/atomic"

// Async is the struct returned from async commands.
type Async struct {
	hdl     Handler
	cmd     Command
	data    any
	handled *uint32
	pending chan bool
	err     error
}

func newResultAsync(hdl Handler, cmd Command) *Async {
	return &Async{
		hdl:     hdl,
		cmd:     cmd,
		handled: new(uint32),
		pending: make(chan bool, 1),
	}
}

//------Fetch Data------//

// Await for the data from the return of the command
func (res *Async) Await() error {
	if !res.isHandled() {
		<-res.pending
		res.setHandled()
	}
	return res.err
}

// Get retrieves the data from the return of the first command.
func (res *Async) Get() (any, error) {
	if err := res.Await(); err != nil {
		return nil, err
	}
	return res.data, nil
}

//------Internal------//

func (res *Async) done() {
	res.pending <- true
}

func (res *Async) setHandled() {
	atomic.CompareAndSwapUint32(res.handled, 0, 1)
}

func (res *Async) isHandled() bool {
	return atomic.LoadUint32(res.handled) == 1
}

func (res *Async) fail(err error) {
	res.err = err
	res.done()
}

func (res *Async) setReturn(data any) {
	res.data = data
}
