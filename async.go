package command

import "sync"

// Async is the struct returned from async commands.
type Async struct {
	sync.Mutex
	hdl      Handler
	cmd      Command
	data     any
	done     *flag
	pending  chan bool
	listener func(as *Async)
	err      error
}

func newAsync(hdl Handler, cmd Command) *Async {
	return &Async{
		hdl:     hdl,
		cmd:     cmd,
		done:    newFlag(),
		pending: make(chan bool, 1),
	}
}

// Await for the command to be processed.
func (as *Async) Await() (any, error) {
	as.await()
	if as.err != nil {
		return nil, as.err
	}
	return as.data, nil
}

//------Internal------//

func (as *Async) await() {
	if !as.done.enabled() {
		<-as.pending
	}
}

func (as *Async) notifyDone() {
	if as.done.enable() {
		as.pending <- true
		as.notifyListener()
	}
}

func (as *Async) fail(err error) {
	as.err = err
	as.notifyDone()
}

func (as *Async) success(data any) {
	as.data = data
	as.notifyDone()
}

func (as *Async) setListener(listener func(as *Async)) {
	as.Lock()
	as.listener = listener
	as.Unlock()
	if as.done.enabled() {
		as.notifyListener()
	}
}

func (as *Async) notifyListener() {
	as.Lock()
	if as.listener != nil {
		as.listener(as)
	}
	as.Unlock()
}
