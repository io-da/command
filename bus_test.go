package command

import (
	"sync"
	"testing"
	"time"
)

func TestBus_Initialize(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}
	hdl2 := &testHandlerAsync{}

	bus.Initialize(hdl, hdl2)
	if len(bus.handlers) != 2 {
		t.Error("Unexpected number of handlers.")
	}
}

func TestBus_AsyncBuffer(t *testing.T) {
	bus := NewBus()
	bus.QueueBuffer(1000)
	bus.Initialize()
	if cap(bus.asyncCommandsQueue) != 1000 {
		t.Error("Unexpected async command queue capacity.")
	}
}

func TestBus_Handle(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}

	bus.Initialize(hdl)
	bus.Handle(&testCommand1{})
	bus.Handle(&testCommand2{})
	bus.Handle(testCommand3("test"))
}

func TestBus_HandleAsync(t *testing.T) {
	bus := NewBus()
	bus.WorkerPoolSize(4)
	wg := &sync.WaitGroup{}
	hdl := &testHandlerAsync{wg: wg}

	wg.Add(3)
	bus.Initialize(hdl)
	bus.HandleAsync(&testCommand1{})
	bus.HandleAsync(&testCommand2{})
	bus.HandleAsync(testCommand3("test"))

	timeout := time.AfterFunc(time.Second*10, func() {
		t.Fatal("The commands should have been handled by now.")
	})

	wg.Wait()
	timeout.Stop()
}

func TestBus_Shutdown(t *testing.T) {
	bus := NewBus()
	hdl := &testHandler{}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	bus.Initialize(hdl)
	bus.HandleAsync(&testCommand1{})
	time.AfterFunc(time.Nanosecond, func() {
		// graceful shutdown
		bus.Shutdown()
		wg.Done()
	})

	for i := 0; i < 1000; i++ {
		bus.HandleAsync(&testCommand1{})
	}
	wg.Wait()

	if !bus.isShuttingDown() {
		t.Error("The bus should be shutting down.")
	}
}

func TestBus_HandlerOrder(t *testing.T) {
	bus := NewBus()
	wg := &sync.WaitGroup{}

	hdls := make([]Handler, 0, 1000)
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		hdls = append(hdls, &testHandlerOrder{wg: wg, position: uint32(i)})
	}
	bus.Initialize(hdls...)

	cmd := &testHandlerOrderCommand{position: new(uint32), unordered: new(uint32)}
	bus.HandleAsync(cmd)

	timeout := time.AfterFunc(time.Second*10, func() {
		t.Fatal("The commands should have been handled by now.")
	})

	wg.Wait()
	timeout.Stop()
	if cmd.IsUnordered() {
		t.Error("The Handler order MUST be respected.")
	}
}

func BenchmarkBus_Handling1MillionCommands(b *testing.B) {
	bus := NewBus()
	wg := &sync.WaitGroup{}

	bus.Initialize(&testHandlerAsync{wg: wg})
	for n := 0; n < b.N; n++ {
		wg.Add(1000000)
		for i := 0; i < 1000000; i++ {
			bus.Handle(&testCommand1{})
		}
		wg.Wait()
	}
}

func BenchmarkBus_Handling1MillionAsyncCommands(b *testing.B) {
	bus := NewBus()
	wg := &sync.WaitGroup{}

	bus.Initialize(&testHandlerAsync{wg: wg})
	for n := 0; n < b.N; n++ {
		wg.Add(1000000)
		for i := 0; i < 1000000; i++ {
			bus.HandleAsync(&testCommand1{})
		}
		wg.Wait()
	}
}
