package command

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
)

// ------Enums------//
const (
	Unidentified Identifier = iota
	TestCommand1
	TestCommand2
	TestLiteralCommand
	TestErrorCommand
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

type testCommand3 string

func (testCommand3) Identifier() Identifier {
	return TestLiteralCommand
}

type testCommandError struct{}

func (*testCommandError) Identifier() Identifier {
	return TestErrorCommand
}

//------Handlers------//

type testHandler struct {
	identifier Identifier
}

func (hdl *testHandler) Handles() Identifier {
	return hdl.identifier
}

func (hdl *testHandler) Handle(cmd Command) (data any, err error) {
	fibonacci(1000)
	return
}

type testErrorHandler struct{}

func (hdl *testErrorHandler) Handles() Identifier {
	return TestErrorCommand
}

func (hdl *testErrorHandler) Handle(cmd Command) (data any, err error) {
	err = errors.New("command failed")
	return
}

type testAsyncHandler struct {
	wg         *sync.WaitGroup
	identifier Identifier
}

func (hdl *testAsyncHandler) Handles() Identifier {
	return hdl.identifier
}

func (hdl *testAsyncHandler) Handle(cmd Command) (data any, err error) {
	fibonacci(1000)
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
		data = nil
	case *testCommand2:
		data = "ok"
	}
	return data, err
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
}

func newTestLoggerMiddleware(logHandler chan string, testId string) *testLoggerMiddleware {
	return &testLoggerMiddleware{
		logHandler: logHandler,
		testId:     testId,
	}
}

func (hdl *testLoggerMiddleware) HandleInward(cmd Command) error {
	hdl.log(fmt.Sprintf("%s|inward|%d", hdl.testId, cmd.Identifier()))
	return nil
}

func (hdl *testLoggerMiddleware) HandleOutward(cmd Command, data any, err error) error {
	hdl.log(fmt.Sprintf("%s|outward|%d", hdl.testId, cmd.Identifier()))
	return nil
}

func (hdl *testLoggerMiddleware) log(message string) {
	hdl.logHandler <- message
}

type testErrorMiddleware struct {
	inwardFailure  bool
	outwardFailure bool
}

func (hdl *testErrorMiddleware) HandleInward(cmd Command) error {
	if hdl.inwardFailure {
		return errors.New("inward middleware failure")
	}
	return nil
}

func (hdl *testErrorMiddleware) HandleOutward(cmd Command, data any, err error) error {
	if hdl.outwardFailure {
		return errors.New("outward middleware failure")
	}
	return nil
}

//------General------//

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
