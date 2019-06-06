package command

import (
	"sync"
	"sync/atomic"
	"time"
)

//------Commands------//

type testCommand1 struct {
}

type testCommand2 struct {
}

type testCommand3 string

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

//------Handlers------//

type testHandler struct {
}

func (hdl *testHandler) Handle(cmd Command) {
	switch cmd.(type) {
	case *testCommand1, *testCommand2, testCommand3:
		time.Sleep(time.Nanosecond * 200)
	}
}

type testHandlerAsync struct {
	wg *sync.WaitGroup
}

func (hdl *testHandlerAsync) Handle(cmd Command) {
	switch cmd.(type) {
	case *testCommand1, *testCommand2, testCommand3:
		time.Sleep(time.Nanosecond * 200)
		hdl.wg.Done()
	}
}

type testHandlerOrder struct {
	wg       *sync.WaitGroup
	position uint32
}

func (hdl *testHandlerOrder) Handle(cmd Command) {
	if cmd, listens := cmd.(*testHandlerOrderCommand); listens {
		cmd.HandlerPosition(hdl.position)
		hdl.wg.Done()
	}
}
