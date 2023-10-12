package command

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ------Enums------//
const (
	Unidentified Identifier = iota
	TestCommand1
	TestCommand2
	TestCommand3
	TestCommandError
	TestHandlerOrderCommand
)

//------Commands------//

type testCommand1 struct {
}

func (*testCommand1) Identifier() Identifier {
	return TestCommand1
}

type testCommand2 struct {
}

func (*testCommand2) Identifier() Identifier {
	return TestCommand2
}

type testCommand3 string

func (testCommand3) Identifier() Identifier {
	return TestCommand3
}

type testCommandError struct {
}

func (*testCommandError) Identifier() Identifier {
	return TestCommandError
}

type testHandlerOrderCommand struct {
	position  *uint32
	unordered *uint32
}

func (cmd *testHandlerOrderCommand) HandlerPosition(position uint32) {
	if position != atomic.LoadUint32(cmd.position) {
		atomic.StoreUint32(cmd.unordered, 1)
	}
	atomic.AddUint32(cmd.position, 1)
}
func (cmd *testHandlerOrderCommand) IsUnordered() bool {
	return atomic.LoadUint32(cmd.unordered) == 1
}
func (*testHandlerOrderCommand) Identifier() Identifier {
	return TestHandlerOrderCommand
}

//------Handlers------//

type testHandler struct {
	identifier Identifier
}

func (hdl *testHandler) Handles() Identifier {
	return hdl.identifier
}

func (hdl *testHandler) Handle(cmd Command) (data any, err error) {
	time.Sleep(time.Nanosecond * 200)
	return
}

type testHandlerError struct {
}

func (hdl *testHandlerError) Handles() Identifier {
	return TestCommandError
}

func (hdl *testHandlerError) Handle(cmd Command) (data any, err error) {
	err = errors.New("command failed")
	return
}

type testHandlerAsync struct {
	wg         *sync.WaitGroup
	identifier Identifier
}

func (hdl *testHandlerAsync) Handles() Identifier {
	return hdl.identifier
}

func (hdl *testHandlerAsync) Handle(cmd Command) (data any, err error) {
	time.Sleep(time.Nanosecond * 200)
	hdl.wg.Done()
	return
}

type testHandlerAsyncAwait struct {
	identifier Identifier
}

func (hdl *testHandlerAsyncAwait) Handles() Identifier {
	return hdl.identifier
}

func (hdl *testHandlerAsyncAwait) Handle(cmd Command) (data any, err error) {
	data = "not ok"
	switch any(cmd).(type) {
	case *testCommand1:
		data = nil
	case *testCommand2:
		data = "ok"
	}
	return data, err
}

type testHandlerScheduled struct {
	wg *sync.WaitGroup
}

func (hdl *testHandlerScheduled) Handles() Identifier {
	return TestCommand1
}

func (hdl *testHandlerScheduled) Handle(cmd Command) (data any, err error) {
	time.Sleep(time.Nanosecond * 200)
	hdl.wg.Done()
	return
}

type testHandlerOrder struct {
	wg         *sync.WaitGroup
	position   uint32
	identifier Identifier
}

func (hdl *testHandlerOrder) Handles() Identifier {
	return hdl.identifier
}

func (hdl *testHandlerOrder) Handle(cmd Command) (data any, err error) {
	if cmd, listens := cmd.(*testHandlerOrderCommand); listens {
		cmd.HandlerPosition(hdl.position)
		hdl.wg.Done()
	}
	return
}

//------Error Handlers------//

type storeErrorsHandler struct {
	sync.Mutex
	errs map[Identifier]error
}

func (hdl *storeErrorsHandler) Handle(cmd Command, err error) {
	hdl.Lock()
	hdl.errs[hdl.key(cmd)] = err
	hdl.Unlock()
}

func (hdl *storeErrorsHandler) Error(cmd Command) error {
	hdl.Lock()
	defer hdl.Unlock()
	if err, hasError := hdl.errs[hdl.key(cmd)]; hasError {
		return err
	}
	return nil
}

func (hdl *storeErrorsHandler) key(cmd Command) Identifier {
	if cmd == nil {
		return Unidentified
	}
	return cmd.Identifier()
}
