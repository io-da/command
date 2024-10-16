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
	notify   *flag
	listener func(as *Async)
	err      error
}

func newAsync(hdl Handler, cmd Command) *Async {
	return &Async{
		hdl:     hdl,
		cmd:     cmd,
		done:    newFlag(),
		notify:  newFlag(),
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
		as.notifyListener()
		as.pending <- true
	}
}

func (as *Async) fail(err error) {
	as.Lock()
	as.err = err
	as.notifyDone()
	as.Unlock()
}

func (as *Async) success(data any) {
	as.Lock()
	as.data = data
	as.notifyDone()
	as.Unlock()
}

func (as *Async) setListener(listener func(as *Async)) {
	as.Lock()
	if as.done.enabled() {
		listener(as)
		return
	}
	if as.notify.enable() {
		as.listener = listener
	}
	as.Unlock()

}

func (as *Async) notifyListener() {
	if as.notify.disable() {
		as.listener(as)
	}
}
