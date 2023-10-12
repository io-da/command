package command

import "sync/atomic"

// Async is the struct returned from async commands.
type Async struct {
	hdl     Handler
	cmd     Command
	data    any
	done    *uint32
	pending chan bool
	err     error
}

func newAsync(hdl Handler, cmd Command) *Async {
	return &Async{
		hdl:     hdl,
		cmd:     cmd,
		done:    new(uint32),
		pending: make(chan bool, 1),
	}
}

//------Fetch Data------//

// Await for the data from the return of the command
func (res *Async) Await() error {
	if !res.isDone() {
		<-res.pending
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

func (res *Async) notifyDone() {
	res.setDone()
	res.pending <- true
}

func (res *Async) setDone() {
	atomic.CompareAndSwapUint32(res.done, 0, 1)
}

func (res *Async) isDone() bool {
	return atomic.LoadUint32(res.done) == 1
}

func (res *Async) fail(err error) {
	res.err = err
	res.notifyDone()
}

func (res *Async) success(data any) {
	res.data = data
	res.notifyDone()
}
