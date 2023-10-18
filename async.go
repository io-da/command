package command

// Async is the struct returned from async commands.
type Async struct {
	hdl     Handler
	cmd     Command
	cls     ClosureCommand
	isCls   bool
	data    any
	done    *flag
	pending chan bool
	err     error
}

func newAsync(hdl Handler, cmd Command) *Async {
	return &Async{
		hdl:     hdl,
		cmd:     cmd,
		done:    newFlag(),
		pending: make(chan bool, 1),
	}
}

func newAsyncClosure(cls ClosureCommand) *Async {
	return &Async{
		cls:     cls,
		isCls:   true,
		done:    newFlag(),
		pending: make(chan bool, 1),
	}
}

//------Fetch Data------//

// Await for the command to be processed.
func (res *Async) Await() (any, error) {
	if !res.done.enabled() {
		<-res.pending
	}
	if res.err != nil {
		return nil, res.err
	}
	return res.data, nil
}

//------Internal------//

func (res *Async) notifyDone() {
	if res.done.enable() {
		res.pending <- true
	}
}

func (res *Async) fail(err error) {
	res.err = err
	res.notifyDone()
}

func (res *Async) success(data any) {
	res.data = data
	res.notifyDone()
}
