package command

import (
	"github.com/google/uuid"
	"github.com/io-da/schedule"
	"runtime"
	"sync/atomic"
)

// Bus is the only struct exported and required for the command bus usage.
// The Bus should be instantiated using the NewBus function.
type Bus struct {
	workerPoolSize     int
	queueBuffer        int
	initialized        *uint32
	shuttingDown       *uint32
	workers            *uint32
	handlers           map[Identifier]Handler
	errorHandlers      []ErrorHandler
	inwardMiddlewares  []InwardMiddleware
	outwardMiddlewares []OutwardMiddleware
	asyncCommandsQueue chan *Async
	closed             chan bool
	scheduleProcessor  *scheduleProcessor
}

// NewBus instantiates the Bus struct.
// The Initialization of the Bus is performed separately (Initialize function) for dependency injection purposes.
func NewBus() *Bus {
	bus := &Bus{
		workerPoolSize:     runtime.GOMAXPROCS(0),
		queueBuffer:        100,
		initialized:        new(uint32),
		shuttingDown:       new(uint32),
		workers:            new(uint32),
		handlers:           make(map[Identifier]Handler),
		errorHandlers:      make([]ErrorHandler, 0),
		inwardMiddlewares:  make([]InwardMiddleware, 0),
		outwardMiddlewares: make([]OutwardMiddleware, 0),
		closed:             make(chan bool),
	}
	bus.scheduleProcessor = newScheduleProcessor(bus)
	return bus
}

// SetWorkerPoolSize may optionally be used to tweak the worker pool size for async commands.
// It can only be adjusted *before* the bus is initialized.
// It defaults to the value returned by runtime.GOMAXPROCS(0).
func (bus *Bus) SetWorkerPoolSize(workerPoolSize int) {
	if !bus.isInitialized() {
		bus.workerPoolSize = workerPoolSize
	}
}

// SetQueueBuffer may optionally be used to tweak the buffer size of the async commands queue.
// This value may have high impact on performance depending on the use case.
// It can only be adjusted *before* the bus is initialized.
// It defaults to 100.
func (bus *Bus) SetQueueBuffer(queueBuffer int) {
	if !bus.isInitialized() {
		bus.queueBuffer = queueBuffer
	}
}

// SetErrorHandlers may optionally be used to provide a list of error handlers.
// They will receive any error thrown during the command process.
// Error handlers may only be provided *before* the bus is initialized.
func (bus *Bus) SetErrorHandlers(hdls ...ErrorHandler) {
	if !bus.isInitialized() {
		bus.errorHandlers = hdls
	}
}

// SetInwardMiddlewares may optionally be used to provide a list of inward middlewares.
// They will receive and process every command that is about to be handled.
// *The order the middlewares are provided is always respected*.
// Middlewares may only be provided *before* the bus is initialized.
func (bus *Bus) SetInwardMiddlewares(mdls ...InwardMiddleware) {
	if !bus.isInitialized() {
		bus.inwardMiddlewares = mdls
	}
}

// SetOutwardMiddlewares may optionally be used to provide a list of outward middlewares.
// They will receive and process every command that was handled.
// *The order the middlewares are provided is always respected*.
// Middlewares may only be provided *before* the bus is initialized.
func (bus *Bus) SetOutwardMiddlewares(mdls ...OutwardMiddleware) {
	if !bus.isInitialized() {
		bus.outwardMiddlewares = mdls
	}
}

// Initialize the command bus by providing the list of handlers.
// There can only be one handler per command.
func (bus *Bus) Initialize(hdls ...Handler) error {
	if bus.initialize() {
		for _, hdl := range hdls {
			if _, exists := bus.handlers[hdl.Handles()]; exists {
				return OneHandlerPerCommandError
			}
			bus.handlers[hdl.Handles()] = hdl
		}
		bus.asyncCommandsQueue = make(chan *Async, bus.queueBuffer)
		for i := 0; i < bus.workerPoolSize; i++ {
			bus.workerUp()
			go bus.worker(bus.asyncCommandsQueue, bus.closed)
		}
	}
	return nil
}

// HandleAsync the command using the workers asynchronously.
// It also returns an *Async struct which allows clients to optionally await for the command to be processed.
func (bus *Bus) HandleAsync(cmd Command) (*Async, error) {
	hdl, err := bus.handler(cmd)
	if err != nil {
		return nil, err
	}
	return bus.addToAsyncQueue(hdl, cmd), nil
}

// Handle the command synchronously.
func (bus *Bus) Handle(cmd Command) (any, error) {
	hdl, err := bus.handler(cmd)
	if err != nil {
		return nil, err
	}
	return bus.handle(hdl, cmd)
}

// Schedule allows commands to be scheduled to be executed asynchronously.
func (bus *Bus) Schedule(cmd Command, sch *schedule.Schedule) (*uuid.UUID, error) {
	hdl, err := bus.handler(cmd)
	if err != nil {
		return nil, err
	}
	key := bus.scheduleProcessor.add(newScheduledCommand(hdl, cmd, sch))
	return &key, nil
}

// RemoveScheduled removes previously scheduled commands.
func (bus *Bus) RemoveScheduled(keys ...uuid.UUID) {
	bus.scheduleProcessor.remove(keys...)
}

// Shutdown the command bus gracefully.
// *Async commands accessed while shutting down will be disregarded*.
func (bus *Bus) Shutdown() {
	if atomic.CompareAndSwapUint32(bus.shuttingDown, 0, 1) {
		go bus.shutdown()
	}
}

//-----Private Functions------//

func (bus *Bus) initialize() bool {
	return atomic.CompareAndSwapUint32(bus.initialized, 0, 1)
}

func (bus *Bus) isInitialized() bool {
	return atomic.LoadUint32(bus.initialized) == 1
}

func (bus *Bus) isShuttingDown() bool {
	return atomic.LoadUint32(bus.shuttingDown) == 1
}

func (bus *Bus) worker(asyncCommandsQueue <-chan *Async, closed chan<- bool) {
	for async := range asyncCommandsQueue {
		if async == nil {
			break
		}
		bus.handleAsync(async)
	}
	closed <- true
}

func (bus *Bus) handle(hdl Handler, cmd Command) (data any, err error) {
	for _, inMdl := range bus.inwardMiddlewares {
		if err = inMdl.HandleInward(cmd); err != nil {
			bus.error(cmd, err)
			return
		}
	}
	if data, err = hdl.Handle(cmd); err != nil {
		data = nil
		bus.error(cmd, err)
	}
	for _, outMdl := range bus.outwardMiddlewares {
		if err = outMdl.HandleOutward(cmd, data, err); err != nil {
			bus.error(cmd, err)
			return
		}
	}
	return
}

func (bus *Bus) handleAsync(async *Async) {
	data, err := bus.handle(async.hdl, async.cmd)
	if err != nil {
		async.fail(err)
		return
	}
	async.success(data)
}

func (bus *Bus) addToAsyncQueue(hdl Handler, cmd Command) *Async {
	async := newAsync(hdl, cmd)
	bus.asyncCommandsQueue <- async
	return async
}

func (bus *Bus) workerUp() {
	atomic.AddUint32(bus.workers, 1)
}

func (bus *Bus) workerDown() {
	atomic.AddUint32(bus.workers, ^uint32(0))
}

func (bus *Bus) shutdown() {
	for atomic.LoadUint32(bus.workers) > 0 {
		bus.asyncCommandsQueue <- nil
		<-bus.closed
		bus.workerDown()
	}
	bus.scheduleProcessor.shutdown()
	atomic.CompareAndSwapUint32(bus.initialized, 1, 0)
	atomic.CompareAndSwapUint32(bus.shuttingDown, 1, 0)
}

func (bus *Bus) handler(cmd Command) (hdl Handler, err error) {
	if cmd == nil {
		err = InvalidCommandError
		bus.error(cmd, err)
		return
	}
	if !bus.isInitialized() {
		err = BusNotInitializedError
		bus.error(cmd, err)
		return
	}
	if bus.isShuttingDown() {
		err = BusIsShuttingDownError
		bus.error(cmd, err)
		return
	}
	hdl, ok := bus.handlers[cmd.Identifier()]
	if !ok {
		err = HandlerNotFoundError
		bus.error(cmd, err)
	}
	return
}

func (bus *Bus) error(cmd Command, err error) {
	for _, errHdl := range bus.errorHandlers {
		errHdl.Handle(cmd, err)
	}
}
