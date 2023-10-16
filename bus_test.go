package command

import (
	"github.com/io-da/schedule"
	"sync"
	"testing"
	"time"
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
	wg := &sync.WaitGroup{}
	hdl := &testAsyncHandler{wg: wg, identifier: TestCommand1}
	hdl2 := &testAsyncHandler{wg: wg, identifier: TestLiteralCommand}

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.HandleAsync(&testCommand2{}); err == nil || err != HandlerNotFoundError {
		t.Error("Expected HandlerNotFoundError error.")
	}
	wg.Add(2)
	if _, err := bus.HandleAsync(&testCommand1{}); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.HandleAsync(testCommand3("test")); err != nil {
		t.Fatal(err.Error())
	}

	timeout := time.AfterFunc(time.Second*10, func() {
		t.Fatal("The commands should have been accessed by now.")
	})

	wg.Wait()
	timeout.Stop()
}

func TestBus_HandleAsyncAwait(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	hdl := &testAsyncAwaitHandler{identifier: TestCommand1}
	hdl2 := &testAsyncAwaitHandler{identifier: TestCommand2}

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	res, err := bus.HandleAsync(&testCommand1{})
	if err != nil {
		t.Fatal(err.Error())
	}
	res2, err := bus.HandleAsync(&testCommand2{})
	if err != nil {
		t.Fatal(err.Error())
	}

	timeout := time.AfterFunc(time.Second*10, func() {
		t.Fatal("The commands should have been accessed by now.")
	})
	if err = res.Await(); err != nil {
		t.Fatal(err.Error())
	}
	data, err := res.Get()
	if err != nil {
		t.Fatal(err.Error())
	}
	if data != nil {
		t.Error("The command handler returned unexpected data.")
	}

	if err = res2.Await(); err != nil {
		t.Fatal(err.Error())
	}
	data, err = res2.Get()
	if err != nil {
		t.Fatal(err.Error())
	}
	if data != "ok" {
		t.Error("The command handler returned unexpected data.")
	}

	timeout.Stop()
}

func TestBus_HandleAsyncAwaitFail(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	hdl := &testErrorHandler{}

	if err := bus.Initialize(hdl); err != nil {
		t.Fatal(err.Error())
	}
	cmd := &testCommandError{}
	res, err := bus.HandleAsync(cmd)
	if err != nil {
		t.Fatal(err.Error())
	}

	timeout := time.AfterFunc(time.Second*10, func() {
		t.Fatal("The commands should have been accessed by now.")
	})

	_, err = res.Get()
	if err == nil {
		t.Error("Command handler was expected to throw an error.")
	}
	if err.Error() != "command failed" {
		t.Fatal(err.Error())
	}

	timeout.Stop()
}

func TestBus_HandleScheduled(t *testing.T) {
	bus := NewBus()
	bus.SetWorkerPoolSize(4)
	wg := &sync.WaitGroup{}
	hdl := &testAsyncHandler{wg: wg, identifier: TestCommand1}

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

	timeout := time.AfterFunc(time.Second*5, func() {
		t.Fatal("The commands should have been accessed by now.")
	})

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
	bus.SetInwardMiddlewares(mdl1, mdl2)
	bus.SetOutwardMiddlewares(mdl1, mdl2)

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
		"logger1|inward|1",
		"logger2|inward|1",
		"logger1|outward|1",
		"logger2|outward|1",
		"logger1|inward|2",
		"logger2|inward|2",
		"logger1|outward|2",
		"logger2|outward|2",
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

	logs := make(chan string, 2)
	mdl1 := newTestLoggerMiddleware(logs, "logger1")
	mdl2 := newTestLoggerMiddleware(logs, "logger2")
	mdl3 := &testErrorMiddleware{inwardFailure: true}
	bus.SetInwardMiddlewares(mdl1, mdl3, mdl2)
	bus.SetOutwardMiddlewares(mdl1, mdl3, mdl2)

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	if _, err := bus.Handle(&testCommand1{}); err == nil {
		t.Error("Expected middleware error.")
	}
	if _, err := bus.Handle(&testCommand2{}); err == nil {
		t.Error("Expected middleware error.")
	}

	evaluateMessages(t, []string{
		"logger1|inward|1",
		"logger1|inward|2",
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

	logs := make(chan string, 6)
	mdl1 := newTestLoggerMiddleware(logs, "logger1")
	mdl2 := newTestLoggerMiddleware(logs, "logger2")
	mdl3 := &testErrorMiddleware{outwardFailure: true}
	bus.SetInwardMiddlewares(mdl1, mdl3, mdl2)
	bus.SetOutwardMiddlewares(mdl1, mdl3, mdl2)

	if err := bus.Initialize(hdl, hdl2); err != nil {
		t.Fatal(err.Error())
	}
	res, _ := bus.HandleAsync(&testCommand1{})
	res2, _ := bus.HandleAsync(&testCommand2{})
	if err := res.Await(); err == nil {
		t.Error("Expected middleware error.")
	}
	if err := res2.Await(); err == nil {
		t.Error("Expected middleware error.")
	}
	evaluateMessagesUnordered(t, []string{
		"logger1|inward|1",
		"logger2|inward|1",
		"logger1|outward|1",
		"logger1|inward|2",
		"logger2|inward|2",
		"logger1|outward|2",
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

func evaluateMessagesUnordered(t *testing.T, expectedList []string, messageChan <-chan string) {
	handled := 0
	for message := range messageChan {
		if !listContains(expectedList, message) {
			t.Errorf("Unexpected middleware message %s", message)
			break
		}
		handled++
		if handled == len(expectedList) {
			break
		}
	}
}

func listContains(expectedList []string, message string) bool {
	for _, i := range expectedList {
		if i == message {
			return true
		}
	}
	return false
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
		if !bus.isShuttingDown() {
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

	for bus.isShuttingDown() {
		time.Sleep(time.Microsecond)
	}
	wg.Wait()
}

func BenchmarkBus_Handling1MillionCommands(b *testing.B) {
	bus := NewBus()

	hdls := make([]Handler, 100)
	cmds := make([]Command, 100)
	for i := 0; i < 100; i++ {
		hdls[i] = &testHandler{identifier: Identifier(i)}
		cmds[i] = &testCommand{identifier: Identifier(i)}
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

func BenchmarkBus_Handling1MillionAsyncCommands(b *testing.B) {
	bus := NewBus()
	wg := &sync.WaitGroup{}

	hdls := make([]Handler, 100)
	cmds := make([]Command, 100)
	for i := 0; i < 100; i++ {
		hdls[i] = &testAsyncHandler{wg: wg, identifier: Identifier(i)}
		cmds[i] = &testCommand{identifier: Identifier(i)}
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
