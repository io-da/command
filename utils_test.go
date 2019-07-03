package command

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

//------Commands------//

type testCommand1 struct {
}

func (*testCommand1) Id() []byte {
	return []byte("UUID")
}

type testCommand2 struct {
}

func (*testCommand2) Id() []byte {
	return []byte("UUID")
}

type testCommand3 string

func (testCommand3) Id() []byte {
	return []byte("UUID")
}

type testCommandError struct {
}

func (*testCommandError) Id() []byte {
	return []byte("UUID")
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
func (*testHandlerOrderCommand) Id() []byte {
	return []byte("UUID")
}

//------Handlers------//

type testHandler struct {
}

func (hdl *testHandler) Handle(cmd Command) error {
	switch cmd.(type) {
	case *testCommand1, *testCommand2, testCommand3:
		time.Sleep(time.Nanosecond * 200)
	}
	return nil
}

type testHandlerError struct {
}

func (hdl *testHandlerError) Handle(cmd Command) error {
	switch cmd.(type) {
	case *testCommandError:
		return errors.New("command failed")
	}
	return nil
}

type testHandlerAsync struct {
	wg *sync.WaitGroup
}

func (hdl *testHandlerAsync) Handle(cmd Command) error {
	switch cmd.(type) {
	case *testCommand1, *testCommand2, testCommand3:
		time.Sleep(time.Nanosecond * 200)
		hdl.wg.Done()
	}
	return nil
}

type testHandlerOrder struct {
	wg       *sync.WaitGroup
	position uint32
}

func (hdl *testHandlerOrder) Handle(cmd Command) error {
	if cmd, listens := cmd.(*testHandlerOrderCommand); listens {
		cmd.HandlerPosition(hdl.position)
		hdl.wg.Done()
	}
	return nil
}

//------Error Handlers------//

type storeErrorsHandler struct {
	errs map[string]error
}

func (hdl *storeErrorsHandler) Handle(cmd Command, err error) {
	hdl.errs[string(cmd.Id())] = err
}

func (hdl *storeErrorsHandler) Error(cmd Command) error {
	if err, hasError := hdl.errs[string(cmd.Id())]; hasError {
		return err
	}
	return nil
}
