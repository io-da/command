package command

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"
)

// ------Enums------//

const (
	Unidentified       Identifier = "Unidentified"
	TestCommand1       Identifier = "TestCommand1"
	TestCommand2       Identifier = "TestCommand2"
	TestCommandSlow    Identifier = "TestCommandSlow"
	TestLiteralCommand Identifier = "TestLiteralCommand"
	TestErrorCommand   Identifier = "TestErrorCommand"
)

const (
	handleTimeoutError     = "timeout reached"
	unexpectedDataError    = "unexpected data"
	commandFailedError     = "command failed"
	middlewareInwardError  = "inward middleware failure"
	middlewareOutwardError = "outward middleware failure"
)

//------Commands------//

type testCommand struct {
	identifier Identifier
}

func (cmd *testCommand) Identifier() Identifier {
	return cmd.identifier
}

type testCommand1 struct{}

func (*testCommand1) Identifier() Identifier {
	return TestCommand1
}

type testCommand2 struct{}

func (*testCommand2) Identifier() Identifier {
	return TestCommand2
}

type testCommandSlow struct{}

func (*testCommandSlow) Identifier() Identifier {
	return TestCommandSlow
}

type testCommand3 string

func (testCommand3) Identifier() Identifier {
	return TestLiteralCommand
}

type testCommandError struct{}

func (*testCommandError) Identifier() Identifier {
	return TestErrorCommand
}

type testFakeClosureCommand struct{}

func (*testFakeClosureCommand) Identifier() Identifier {
	return ClosureIdentifier
}

//------Handlers------//

type testHandler struct {
	handles Identifier
}

func (hdl *testHandler) Handles() Identifier {
	return hdl.handles
}

func (hdl *testHandler) Handle(cmd Command) (data any, err error) {
	if _, ok := any(cmd).(*testCommandSlow); ok {
		slowFunc()
	} else {
		fastFunc()
	}
	return
}

type testErrorHandler struct{}

func (hdl *testErrorHandler) Handles() Identifier {
	return TestErrorCommand
}

func (hdl *testErrorHandler) Handle(cmd Command) (data any, err error) {
	err = errors.New(commandFailedError)
	return
}

type testAsyncHandler struct {
	wg      *sync.WaitGroup
	handles Identifier
}

func (hdl *testAsyncHandler) Handles() Identifier {
	return hdl.handles
}

func (hdl *testAsyncHandler) Handle(cmd Command) (data any, err error) {
	if _, ok := any(cmd).(*testCommandSlow); ok {
		slowFunc()
	} else {
		fastFunc()
	}
	hdl.wg.Done()
	return
}

type testAsyncAwaitHandler struct {
	identifier Identifier
}

func (hdl *testAsyncAwaitHandler) Handles() Identifier {
	return hdl.identifier
}

func (hdl *testAsyncAwaitHandler) Handle(cmd Command) (data any, err error) {
	data = "not ok"
	switch any(cmd).(type) {
	case *testCommand1:
		fastFunc()
		data = nil
	case *testCommand2:
		fastFunc()
		data = "ok"
	case *testCommandSlow:
		slowFunc()
		data = "slow ok"
	}
	return data, err
}

type testClosureHandler struct {
}

func (hdl *testClosureHandler) Handles() Identifier {
	return ClosureIdentifier
}

func (hdl *testClosureHandler) Handle(cmd Command) (data any, err error) {
	if cmd, ok := any(cmd).(Closure); ok {
		data, err = cmd()
		if err != nil {
			return
		}
		data, ok := data.(int)
		if !ok {
			return nil, errors.New(unexpectedDataError)
		}
		return 2 + int(data), nil
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

// ------Middlewares------//

type testLoggerMiddleware struct {
	logHandler chan string
	testId     string
	logged     int
}

func newTestLoggerMiddleware(logHandler chan string, testId string) *testLoggerMiddleware {
	return &testLoggerMiddleware{
		logHandler: logHandler,
		testId:     testId,
	}
}

func (hdl *testLoggerMiddleware) Handle(cmd Command, next Next) (data any, err error) {
	hdl.log(fmt.Sprintf("%s|inward|%s", hdl.testId, cmd.Identifier()))
	data, err = next(cmd)
	hdl.log(fmt.Sprintf("%s|outward|%s", hdl.testId, cmd.Identifier()))
	return
}

func (hdl *testLoggerMiddleware) log(message string) {
	hdl.logHandler <- message
	hdl.logged++
}

type testErrorMiddleware struct {
	inwardFailure  bool
	outwardFailure bool
}

func (hdl *testErrorMiddleware) Handle(cmd Command, next Next) (data any, err error) {
	if hdl.inwardFailure {
		return nil, errors.New(middlewareInwardError)
	}
	data, err = next(cmd)
	if hdl.outwardFailure {
		return nil, errors.New(middlewareOutwardError)
	}
	return
}

type testMiddleware struct{}

func (hdl *testMiddleware) Handle(cmd Command, next Next) (data any, err error) {
	fastFunc()
	return next(cmd)
}

//------General------//

var fastFunc = func() { fibonacci(100) }
var slowFunc = func() { fibonacci(100000) }

func fibonacci(n uint) *big.Int {
	if n < 2 {
		return big.NewInt(int64(n))
	}
	a, b := big.NewInt(0), big.NewInt(1)
	for n--; n > 0; n-- {
		a.Add(a, b)
		a, b = b, a
	}

	return b
}

func setupHandleTimeout(t *testing.T) *time.Timer {
	return time.AfterFunc(time.Second*10, func() {
		t.Fatal(handleTimeoutError)
	})
}
