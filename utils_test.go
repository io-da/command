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

func (*testCommand1) ID() []byte {
	return []byte("UUID")
}

type testCommand2 struct {
}

func (*testCommand2) ID() []byte {
	return []byte("UUID")
}

type testCommand3 string

func (testCommand3) ID() []byte {
	return []byte("UUID")
}

type testCommandError struct {
}

func (*testCommandError) ID() []byte {
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
func (*testHandlerOrderCommand) ID() []byte {
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
	sync.Mutex
	errs map[string]error
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

func (hdl *storeErrorsHandler) key(cmd Command) string {
	if cmd == nil {
		return "nil"
	}
	return string(cmd.ID())
}
