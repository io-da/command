package command

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/io-da/schedule"
)

func TestBus_Initialize(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{TestCommand1}
	hdl2 := &testHandler{TestCommand2}
	hdlRepeated := &testHandler{TestCommand1}

	if err := bus.Initialize(hdl, hdl2, hdlRepeated); err == nil || err != OneHandlerPerCommandError {
		t.Error("Expected OneHandlerPerCommandError error.")
	}

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	if len(bus.handlers) != 2 {
		t.Error("Unexpected number of handlers.")
	}
}

func TestBus_AsyncBuffer(t *testing.T) {
	bus := NewBus()
	bus.SetQueueBuffer(1000)
	if err := bus.Initialize(); err != nil {
		t.Fatal(err.Error())
	}

	if cap(bus.asyncCommandsQueue) != 1000 {
		t.Error("Unexpected async command queue capacity.")
	}
}

func TestBus_Handle(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{TestCommand1}
	hdl2 := &testHandler{TestLiteralCommand}

	hdlWErr := &testErrorHandler{}
	errHdl := &storeErrorsHandler{
		errs: make(map[Identifier]error),
	}
	bus.SetErrorHandlers(errHdl)

	if _, err := bus.Handle(nil); err == nil || err != InvalidCommandError {
		t.Error("Expected InvalidCommandError error.")
	}

	cmd := &testCommand1{}
	if _, err := bus.Handle(cmd); err == nil || err != BusNotInitializedError {
		t.Error("Expected BusNotInitializedError error.")
	}

	if err := bus.Initialize(hdl, hdl2, hdlWErr); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testCommand1{}); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testCommand2{}); err == nil || err != HandlerNotFoundError {
		t.Error("Expected HandlerNotFoundError error.")
	}
	if _, err := bus.Handle(testCommand3("test")); err != nil {
		t.Fatal(err.Error())
	}

	errCmd := &testCommandError{}
	if _, err := bus.Handle(errCmd); err == nil {
		t.Error("Command handler was expected to throw an error.")
	}
}

func TestBus_HandleAsync(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	hdl := &testAsyncAwaitHandler{identifier: TestCommand1}
	hdl2 := &testAsyncAwaitHandler{identifier: TestCommand2}

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.HandleAsync(&testCommandSlow{}); err == nil || err != HandlerNotFoundError {
		t.Error("Expected HandlerNotFoundError error.")
	}
	as, err := bus.HandleAsync(&testCommand1{})
	if err != nil {
		t.Fatal(err.Error())
	}
	as2, err := bus.HandleAsync(&testCommand2{})
	if err != nil {
		t.Fatal(err.Error())
	}

	timeout := setupHandleTimeout(t)
	if _, err = as.Await(); err != nil {
		t.Fatal(err.Error())
	}
	data, err := as.Await()
	if err != nil {
		t.Fatal(err.Error())
	}
	if data != nil {
		t.Error(unexpectedDataError)
	}

	if _, err = as2.Await(); err != nil {
		t.Fatal(err.Error())
	}
	data, err = as2.Await()
	if err != nil {
		t.Fatal(err.Error())
	}
	if data != "ok" {
		t.Error(unexpectedDataError)
	}

	timeout.Stop()
}

func TestBus_HandleAsyncList(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	hdl := &testAsyncAwaitHandler{identifier: TestCommand1}
	hdl2 := &testAsyncAwaitHandler{identifier: TestCommand2}

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}

	if _, err := bus.HandleAsyncList(&testCommand1{}, &testCommand2{}, &testCommandSlow{}); err == nil || err != HandlerNotFoundError {
		t.Error("Expected HandlerNotFoundError error.")
	}

	asl, err := bus.HandleAsyncList(&testCommand1{}, &testCommand2{}, Closure(func() (data any, err error) {
		return "bar", nil
	}))
	if err != nil {
		t.Fatal(err.Error())
	}
	as3, err := bus.HandleAsync(&testCommand2{})
	if err != nil {
		t.Fatal(err.Error())
	}

	// simulate scenario of awating an already done command.
	_, err = as3.Await()
	if err != nil {
		t.Fatal(err.Error())
	}

	asl.Push(as3)
	timeout := setupHandleTimeout(t)
	data, err := asl.Await()
	if err != nil {
		t.Fatal(err.Error())
	}
	if data[0] != nil || data[1] != "ok" || data[2] != "bar" || data[3] != "ok" {
		t.Error(unexpectedDataError)
	}
	timeout.Stop()
}

func TestBus_HandleClosure(t *testing.T) {
	bus := NewBus()

	if err := bus.Initialize(); err != nil {
		t.Fatal(err.Error())
	}
	as, _ := bus.HandleAsync(Closure(func() (data any, err error) {
		return "foo bar", nil
	}))

	data, err := as.Await()
	if err != nil {
		t.Fatal(err.Error())
	}
	data, ok := data.(string)
	if !ok || data != "foo bar" {
		t.Error(unexpectedDataError)
	}
}

func TestBus_HandleClosureCustomHandler(t *testing.T) {
	bus := NewBus()

	hdl := &testClosureHandler{}
	if err := bus.Initialize(hdl); err != nil {
		t.Fatal(err.Error())
	}
	as, _ := bus.HandleAsync(Closure(func() (data any, err error) {
		return 2, nil
	}))

	data, err := as.Await()
	if err != nil {
		t.Fatal(err.Error())
	}
	data, ok := data.(int)
	if !ok || data != 4 {
		t.Error(unexpectedDataError)
	}
}

func TestBus_HandleAsyncAwaitError(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	hdl := &testErrorHandler{}

	if err := bus.Initialize(hdl); err != nil {
		t.Fatal(err.Error())
	}
	cmd := &testCommandError{}
	as, err := bus.HandleAsync(cmd)
	if err != nil {
		t.Fatal(err.Error())
	}

	timeout := setupHandleTimeout(t)
	_, err = as.Await()
	if err == nil {
		t.Error("Command handler was expected to throw an error.")
	}
	if err.Error() != commandFailedError {
		t.Fatal(err.Error())
	}

	timeout.Stop()
}

func TestBus_HandleAsyncListAwaitError(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	hdl := &testErrorHandler{}

	if err := bus.Initialize(hdl); err != nil {
		t.Fatal(err.Error())
	}

	asl := NewAsyncList()
	_, err := asl.Await()
	if err == nil {
		t.Error("Async list was expected to throw an error.")
	}
	if err != EmptyAwaitListError {
		t.Fatal("Expected EmptyAwaitListError error.")
	}
	cmd := &testCommandError{}
	as, err := bus.HandleAsync(cmd)
	if err != nil {
		t.Fatal(err.Error())
	}
	asl.Push(as)
	_, err = asl.Await()
	if err == nil {
		t.Error("Command handler was expected to throw an error.")
	}
	if err.Error() != commandFailedError {
		t.Fatal(err.Error())
	}
}

func TestBus_HandleClosureError(t *testing.T) {
	bus := NewBus()

	if err := bus.Initialize(); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testFakeClosureCommand{}); err == nil || err != InvalidClosureCommandError {
		t.Fatal("Expected InvalidClosureCommandError error.")
	}
}

func TestBus_HandleScheduled(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	wg := &sync.WaitGroup{}
	hdl := &testAsyncHandler{wg: wg, handles: TestCommand1}

	_, err := bus.Schedule(&testCommand1{}, schedule.At(time.Now()))
	if err == nil || err != BusNotInitializedError {
		t.Error("Expected BusNotInitializedError error.")
	} else if err.Error() != "command: the bus is not initialized" {
		t.Error("Unexpected BusNotInitializedError message.")
	}
	if err := bus.Initialize(hdl); err != nil {
		t.Fatal(err.Error())
	}

	wg.Add(1)
	if _, err = bus.Schedule(&testCommand1{}, schedule.At(time.Now())); err != nil {
		t.Fatal(err.Error())
	}

	wg.Add(100)
	sch := schedule.At(time.Now())
	sch.AddCron(schedule.Cron().OnMilliseconds(schedule.Between(0, 998).Every(2)))
	uuid1, _ := bus.Schedule(&testCommand1{}, sch)

	for i := 0; i < 100; i++ {
		bus.scheduleProcessor.trigger()
	}

	timeout := setupHandleTimeout(t)
	wg.Wait()
	bus.RemoveScheduled(*uuid1)
	if len(bus.scheduleProcessor.scheduledCommands) > 0 {
		t.Error("The scheduled commands should be empty.")
	}
	timeout.Stop()
}

func TestBus_HandleMiddleware(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{TestCommand1}
	hdl2 := &testHandler{TestCommand2}

	errHdl := &storeErrorsHandler{
		errs: make(map[Identifier]error),
	}
	bus.SetErrorHandlers(errHdl)

	logs := make(chan string, 8)
	mdl1 := newTestLoggerMiddleware(logs, "logger1")
	mdl2 := newTestLoggerMiddleware(logs, "logger2")
	bus.SetMiddlewares(mdl1, mdl2)

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testCommand1{}); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testCommand2{}); err != nil {
		t.Fatal(err.Error())
	}
	evaluateMessages(t, []string{
		"logger1|inward|TestCommand1",
		"logger2|inward|TestCommand1",
		"logger2|outward|TestCommand1",
		"logger1|outward|TestCommand1",
		"logger1|inward|TestCommand2",
		"logger2|inward|TestCommand2",
		"logger2|outward|TestCommand2",
		"logger1|outward|TestCommand2",
	}, logs)
}

func TestBus_HandleMiddlewareInwardError(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{TestCommand1}
	hdl2 := &testHandler{TestCommand2}

	errHdl := &storeErrorsHandler{
		errs: make(map[Identifier]error),
	}
	bus.SetErrorHandlers(errHdl)

	logs := make(chan string, 4)
	mdl1 := newTestLoggerMiddleware(logs, "logger1")
	mdl2 := newTestLoggerMiddleware(logs, "logger2")
	mdl3 := &testErrorMiddleware{inwardFailure: true}
	bus.SetMiddlewares(mdl1, mdl3, mdl2)

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testCommand1{}); err == nil || err.Error() != middlewareInwardError {
		t.Error("Expected middlewareInwardError error.")
	}
	if _, err := bus.Handle(&testCommand2{}); err == nil || err.Error() != middlewareInwardError {
		t.Error("Expected middlewareInwardError error.")
	}
	if mdl1.logged+mdl2.logged != 4 {
		t.Error("Expected 4 logged messages.")
	}

	evaluateMessages(t, []string{
		"logger1|inward|TestCommand1",
		"logger1|outward|TestCommand1",
		"logger1|inward|TestCommand2",
		"logger1|outward|TestCommand2",
	}, logs)
}

func TestBus_HandleMiddlewareOutwardError(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{TestCommand1}
	hdl2 := &testHandler{TestCommand2}

	errHdl := &storeErrorsHandler{
		errs: make(map[Identifier]error),
	}
	bus.SetErrorHandlers(errHdl)

	logs := make(chan string, 8)
	mdl1 := newTestLoggerMiddleware(logs, "logger1")
	mdl2 := newTestLoggerMiddleware(logs, "logger2")
	mdl3 := &testErrorMiddleware{outwardFailure: true}
	bus.SetMiddlewares(mdl1, mdl3, mdl2)

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testCommand1{}); err == nil || err.Error() != middlewareOutwardError {
		t.Error("Expected middlewareOutwardError error.")
	}
	if _, err := bus.Handle(&testCommand2{}); err == nil || err.Error() != middlewareOutwardError {
		t.Error("Expected middlewareOutwardError error.")
	}
	if mdl1.logged+mdl2.logged != 8 {
		t.Error("Expected 8 logged messages.")
	}

	evaluateMessages(t, []string{
		"logger1|inward|TestCommand1",
		"logger2|inward|TestCommand1",
		"logger2|outward|TestCommand1",
		"logger1|outward|TestCommand1",
		"logger1|inward|TestCommand2",
		"logger2|inward|TestCommand2",
		"logger2|outward|TestCommand2",
		"logger1|outward|TestCommand2",
	}, logs)
}

func evaluateMessages(t *testing.T, expectedList []string, messageChan <-chan string) {
	step := 0
	for message := range messageChan {
		expected := expectedList[step]
		if expected != message {
			t.Errorf("Expected middleware message %s, got %s", expected, message)
			break
		}
		step++
		if step == len(expectedList) {
			break
		}
	}
}

func TestBus_Shutdown(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{TestCommand1}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	bus.SetWorkerPoolSize(1337)
	if err := bus.Initialize(hdl); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.HandleAsync(&testCommand1{}); err != nil {
		t.Fatal(err.Error())
	}
	bus.Shutdown()
	for i := 0; i < 1000; i++ {
		if _, err := bus.HandleAsync(&testCommand1{}); err == nil || err != BusIsShuttingDownError {
			t.Fatal(err.Error())
		}
	}
	go func() {
		// graceful shutdown
		if !bus.shuttingDown.enabled() {
			t.Error("The bus should be shutting down.")
		}
		_, err := bus.Handle(&testCommand1{})
		if err == nil || err != BusIsShuttingDownError {
			t.Error("Expected BusIsShuttingDownError error.")
		} else if err.Error() != "command: the bus is shutting down" {
			t.Error("Unexpected BusIsShuttingDownError message.")
		}
		wg.Done()
	}()

	for bus.shuttingDown.enabled() {
		time.Sleep(time.Microsecond)
	}
	wg.Wait()
}

func BenchmarkBus_Handle(b *testing.B) {
	bus := NewBus()

	hdls := make([]Handler, 100)
	cmds := make([]Command, 100)
	for i := 0; i < 100; i++ {
		identifier := Identifier(strconv.Itoa(i))
		hdls[i] = &testHandler{identifier}
		cmds[i] = &testCommand{identifier}
	}
	if err := bus.Initialize(hdls...); err != nil {
		b.Fatal(err.Error())
	}
	rotation := 0
	for n := 0; n < b.N; n++ {
		_, _ = bus.Handle(cmds[rotation])
		if rotation++; rotation >= 100 {
			rotation = 0
		}
	}
}

func BenchmarkBus_HandleAsync(b *testing.B) {
	bus := NewBus()
	wg := &sync.WaitGroup{}

	hdls := make([]Handler, 100)
	cmds := make([]Command, 100)
	for i := 0; i < 100; i++ {
		identifier := Identifier(strconv.Itoa(i))
		hdls[i] = &testAsyncHandler{wg, identifier}
		cmds[i] = &testCommand{identifier}
	}
	if err := bus.Initialize(hdls...); err != nil {
		b.Fatal(err.Error())
	}
	wg.Add(b.N)
	rotation := 0
	for n := 0; n < b.N; n++ {
		_, _ = bus.HandleAsync(cmds[rotation])
		if rotation++; rotation >= 100 {
			rotation = 0
		}
	}
	wg.Wait()
}

func BenchmarkBus_HandleWithMiddleware(b *testing.B) {
	bus := NewBus()

	hdls := make([]Handler, 100)
	cmds := make([]Command, 100)
	for i := 0; i < 100; i++ {
		identifier := Identifier(strconv.Itoa(i))
		hdls[i] = &testHandler{identifier}
		cmds[i] = &testCommand{identifier}
	}
	bus.SetMiddlewares(&testMiddleware{})
	if err := bus.Initialize(hdls...); err != nil {
		b.Fatal(err.Error())
	}
	rotation := 0
	for n := 0; n < b.N; n++ {
		_, _ = bus.Handle(cmds[rotation])
		if rotation++; rotation >= 100 {
			rotation = 0
		}
	}
}

func BenchmarkBus_HandleAsyncWithMiddleware(b *testing.B) {
	bus := NewBus()
	wg := &sync.WaitGroup{}

	hdls := make([]Handler, 100)
	cmds := make([]Command, 100)
	for i := 0; i < 100; i++ {
		identifier := Identifier(strconv.Itoa(i))
		hdls[i] = &testAsyncHandler{wg, identifier}
		cmds[i] = &testCommand{identifier}
	}
	bus.SetMiddlewares(&testMiddleware{})
	if err := bus.Initialize(hdls...); err != nil {
		b.Fatal(err.Error())
	}
	wg.Add(b.N)
	rotation := 0
	for n := 0; n < b.N; n++ {
		_, _ = bus.HandleAsync(cmds[rotation])
		if rotation++; rotation >= 100 {
			rotation = 0
		}
	}
	wg.Wait()
}

func BenchmarkBus_Fibonacci(b *testing.B) {
	for n := 0; n < b.N; n++ {
		fastFunc()
	}
}
